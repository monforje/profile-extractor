package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"profile-extractor/internal/api"
	"profile-extractor/internal/interview"
	"profile-extractor/internal/prompts"
	"profile-extractor/internal/schema"
	"profile-extractor/internal/validator"

	"github.com/joho/godotenv"
)

func main() {
	// Загрузка переменных окружения
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY not found in environment")
	}

	// Чтение YAML схемы
	yamlContent, err := ioutil.ReadFile("config/dictionary.yaml")
	if err != nil {
		log.Fatal("Error reading dictionary.yaml:", err)
	}

	// Парсинг схемы
	schemaFields, err := schema.ParseYAMLSchema(yamlContent)
	if err != nil {
		log.Fatal("Error parsing schema:", err)
	}

	log.Printf("Loaded schema with %d fields", len(schemaFields))

	// Чтение JSON файла интервью
	interviewPath := "input/interview.json"
	if len(os.Args) > 1 {
		interviewPath = os.Args[1]
	}

	interviewData, err := ioutil.ReadFile(interviewPath)
	if err != nil {
		log.Fatal("Error reading interview file:", err)
	}

	// Парсинг интервью
	interviewObj, err := interview.ParseInterviewJSON(interviewData)
	if err != nil {
		log.Fatal("Error parsing interview JSON:", err)
	}

	log.Printf("Loaded interview: %s", interviewObj.InterviewID)

	// Извлечение текста из ответов интервью
	// Можно выбрать разные способы извлечения:

	// 1. Все ответы подряд
	// userText := interviewObj.ExtractAllAnswers()

	// 2. Ответы с контекстом вопросов (рекомендуется)
	userText := interviewObj.ExtractContextualAnswers()

	// 3. Ответы по блокам (для дополнительной обработки)
	// blockAnswers := interviewObj.ExtractAnswersByBlock()

	log.Printf("Extracted text length: %d characters", len(userText))
	log.Println("Sample extracted text (first 200 chars):")
	if len(userText) > 200 {
		log.Println(userText[:200] + "...")
	} else {
		log.Println(userText)
	}

	// Создание клиента API
	client := api.NewOpenAIClient(apiKey)

	// Этап 1: Извлечение данных
	log.Println("\nStep 1: Extracting profile data from interview...")
	extractionPrompt := prompts.GenerateExtractionPrompt(schemaFields, userText)

	log.Println("Generated extraction prompt:")
	log.Println("---")
	log.Println(extractionPrompt[:500] + "...")
	log.Println("---")

	profileJSON, err := client.ExtractProfile(extractionPrompt)
	if err != nil {
		log.Fatal("Error extracting profile:", err)
	}

	log.Println("Extracted profile:")
	log.Println(profileJSON)

	// Этап 2: Валидация и очистка
	log.Println("\nStep 2: Validating and cleaning profile...")
	validationPrompt := prompts.GenerateValidationPrompt(profileJSON)

	validatedJSON, err := client.ExtractProfile(validationPrompt)
	if err != nil {
		log.Fatal("Error validating profile:", err)
	}

	log.Println("Validated profile:")
	log.Println(validatedJSON)

	// Финальная проверка структуры
	if err := validator.ValidateProfileJSON(validatedJSON, schemaFields); err != nil {
		log.Printf("Validation warning: %v", err)
	}

	// Форматирование JSON для читаемости
	var formatted map[string]interface{}
	json.Unmarshal([]byte(validatedJSON), &formatted)

	// Добавление метаданных интервью
	metadata := interviewObj.GetInterviewMetadata()
	formatted["_metadata"] = map[string]interface{}{
		"source_interview": metadata,
		"processing_info": map[string]interface{}{
			"schema_version":    "1.0",
			"extraction_method": "contextual_answers",
			"text_length":       len(userText),
		},
	}

	prettyJSON, _ := json.MarshalIndent(formatted, "", "  ")

	// Создание папки output если не существует
	os.MkdirAll("output", 0755)

	// Сохранение результата с ID интервью в имени файла
	outputFileName := fmt.Sprintf("output/profile_%s.json", interviewObj.InterviewID)
	err = ioutil.WriteFile(outputFileName, prettyJSON, 0644)
	if err != nil {
		log.Fatal("Error saving profile:", err)
	}

	fmt.Printf("\n✅ Профиль успешно создан из интервью и сохранен в %s!\n", outputFileName)
	fmt.Println("\nМетаданные интервью:")
	metadataJSON, _ := json.MarshalIndent(metadata, "", "  ")
	fmt.Println(string(metadataJSON))

	fmt.Println("\nРезультат:")
	fmt.Println(string(prettyJSON))
}
