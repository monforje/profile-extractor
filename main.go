package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"profile-extractor/internal/api"
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

	// Пример пользовательского текста
	userText := `Я студент, учусь в ДВФУ. Живу во Владивостоке. Пишу на Go и немного на gRPC. Также играю в ДНД за гоблина барда, а еще люблю и профессилнально заниматься сексом в позе наездницы.`

	// Создание клиента API
	client := api.NewOpenAIClient(apiKey)

	// Этап 1: Извлечение данных
	log.Println("Step 1: Extracting profile data...")
	extractionPrompt := prompts.GenerateExtractionPrompt(schemaFields, userText)

	log.Println("Generated extraction prompt:")
	log.Println("---")
	log.Println(extractionPrompt)
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
	prettyJSON, _ := json.MarshalIndent(formatted, "", "  ")

	// Создание папки output если не существует
	os.MkdirAll("output", 0755)

	// Сохранение результата
	err = ioutil.WriteFile("output/profile.json", prettyJSON, 0644)
	if err != nil {
		log.Fatal("Error saving profile:", err)
	}

	fmt.Println("\n✅ Профиль успешно создан и сохранен в output/profile.json!")
	fmt.Println("\nРезультат:")
	fmt.Println(string(prettyJSON))
}
