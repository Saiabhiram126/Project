package main

import (
	"bytes"
	"context"
	"encoding/json" // ✅ Required for JSON parsing
	"fmt"
	"log"
	"net/http" // ✅ Required for HTTP requests
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/jackc/pgx/v4"
	"github.com/joho/godotenv"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))
var dbConnection *pgx.Conn
var clients = make(map[*websocket.Conn]bool) // Store connected WebSocket clients

// User struct represents a user in the database
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// Task struct represents a task in the database
type Task struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Assigned  string    `json:"assigned"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// LoginRequest struct for login requests
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	// Database connection
	dbURL := os.Getenv("DATABASE_URL")
	dbConnection, err = pgx.Connect(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	app := fiber.New()
	app.Use(cors.New())
	app.Use(logger.New())

	// ✅ Setup WebSockets
	setupWebSocket(app)

	// API Routes
	app.Post("/register", registerUser)
	app.Post("/login", LoginHandler)
	app.Post("/task", createTask)
	app.Get("/tasks", getTasks)
	app.Post("/task/suggest", suggestTask) // ✅ Added missing suggestTask route

	// Start server
	log.Fatal(app.Listen(":8080"))
}

// Register a new user
func registerUser(c *fiber.Ctx) error {
	var user User
	if err := c.BodyParser(&user); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	_, err := dbConnection.Exec(context.Background(),
		"INSERT INTO users (username, password) VALUES ($1, $2)",
		user.Username, user.Password)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to register user"})
	}

	return c.JSON(fiber.Map{"message": "User registered successfully"})
}

// Login handler with JWT authentication
func LoginHandler(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Bad request"})
	}

	user, err := GetUserFromDB(req.Username)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}

	// Validate password
	if user.Password != req.Password {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}

	// Generate JWT Token
	token, err := GenerateJWT(user)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate token"})
	}

	return c.JSON(fiber.Map{"token": token})
}

// Fetch user from database
func GetUserFromDB(username string) (User, error) {
	var user User
	err := dbConnection.QueryRow(context.Background(),
		"SELECT id, username, password FROM users WHERE username=$1", username).
		Scan(&user.ID, &user.Username, &user.Password)

	if err != nil {
		return User{}, err
	}

	return user, nil
}

// Generate JWT Token
func GenerateJWT(user User) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["user_id"] = user.ID
	claims["username"] = user.Username
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()

	t, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return t, nil
}

// Create a new task
func createTask(c *fiber.Ctx) error {
	var task Task
	if err := c.BodyParser(&task); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	_, err := dbConnection.Exec(context.Background(),
		"INSERT INTO tasks (title, assigned, status, created_at) VALUES ($1, $2, $3, $4)",
		task.Title, task.Assigned, task.Status, time.Now())

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create task"})
	}

	broadcastTaskUpdate(task) // ✅ Broadcast task update via WebSocket

	return c.JSON(fiber.Map{"message": "Task created successfully"})
}

// Retrieve all tasks
func getTasks(c *fiber.Ctx) error {
	rows, err := dbConnection.Query(context.Background(), "SELECT id, title, assigned, status, created_at FROM tasks")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to retrieve tasks"})
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		err := rows.Scan(&task.ID, &task.Title, &task.Assigned, &task.Status, &task.CreatedAt)
		if err != nil {
			continue
		}
		tasks = append(tasks, task)
	}

	fmt.Println("Returning tasks:", tasks) // ✅ Debugging log

	// ✅ Return an empty array instead of `null`
	if len(tasks) == 0 {
		return c.JSON([]Task{})
	}

	return c.JSON(tasks)
}

// WebSocket setup
func setupWebSocket(app *fiber.App) {
	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		clients[c] = true
		defer func() {
			delete(clients, c)
			c.Close()
		}()

		for {
			messageType, msg, err := c.ReadMessage()
			if err != nil {
				break
			}

			for client := range clients {
				if err := client.WriteMessage(messageType, msg); err != nil {
					delete(clients, client)
					client.Close()
				}
			}
		}
	}))
}

// Broadcast task updates to WebSocket clients
func broadcastTaskUpdate(task Task) {
	for client := range clients {
		err := client.WriteJSON(task)
		if err != nil {
			delete(clients, client)
			client.Close()
		}
	}
}
func suggestTask(c *fiber.Ctx) error {
	var req struct {
		Prompt string `json:"prompt"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	response, err := callOpenAI(req.Prompt, openaiAPIKey)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to get AI suggestions"})
	}

	return c.JSON(fiber.Map{"suggestion": response})
}
func callOpenAI(prompt string, apiKey string) (string, error) {
	url := "https://api.openai.com/v1/completions"
	data := map[string]interface{}{
		"model":      "gpt-4",
		"prompt":     prompt,
		"max_tokens": 100,
	}
	jsonData, _ := json.Marshal(data)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)

	// ✅ Safely extract the text response
	choices, ok := res["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("invalid response from OpenAI")
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid response structure")
	}

	text, ok := choice["text"].(string)
	if !ok {
		return "", fmt.Errorf("text not found in response")
	}

	return text, nil
}
