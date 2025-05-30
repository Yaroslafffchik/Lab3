package main

import (
	"database/sql"
	"log"
	_ "server/docs" // Не забудьте, что 'server/docs' должен быть правильным путём до сгенерированного файла

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger" // fiber swagger middleware
	"github.com/lib/pq"          // PostgreSQL driver
	_ "github.com/lib/pq"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

var db *sql.DB

func initDB() {
	var err error
	// Убедитесь, что PostgreSQL запущен и доступен с этими параметрами
	db, err = sql.Open("postgres", "host=localhost port=5432 user=postgres password=CAXAPOK2005ya dbname=db sslmode=disable")
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("Ошибка пинга базы данных: %v", err)
	}
	log.Println("Успешное подключение к базе данных PostgreSQL")

	_, err = db.Exec(`
    CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    description TEXT,
    categories TEXT[] -- Массив строк для хранения категорий
);
`)
	if err != nil {
		log.Fatalf("Ошибка создания таблицы: %v", err)
	}
	log.Println("Таблица 'products' проверена/создана")
}

type Product struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Price       float64  `json:"price"`
	Description string   `json:"description"`
	Categories  []string `json:"categories"`
}

type CreateProductRequest struct {
	Name        string   `json:"name"`
	Price       float64  `json:"price"`
	Description string   `json:"description"`
	Categories  []string `json:"categories"`
}

type UpdateProductRequest struct {
	Name        string   `json:"name"`
	Price       float64  `json:"price"`
	Description string   `json:"description"`
	Categories  []string `json:"categories"`
}

// @Summary Получение списка всех продуктов
// @Tags Products
// @Accept json
// @Produce json
// @Success 200 {array} Product "Успешный ответ"
// @Failure 500 {object} ErrorResponse "Ошибка на сервере"
// @Router /api/products [get]
func getProducts(c *fiber.Ctx) error {
	rows, err := db.Query("SELECT id, name, price, description, categories FROM products")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var product Product
		if err := rows.Scan(&product.ID, &product.Name, &product.Price, &product.Description, pq.Array(&product.Categories)); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
		}
		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.JSON(products)
}

// @Summary Добавить один или несколько продуктов
// @Tags Products
// @Accept json
// @Produce json
// @Param products body []CreateProductRequest true "Данные продуктов"
// @Success 200 {array} Product "Продукты успешно добавлены"
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 500 {object} ErrorResponse "Ошибка на сервере"
// @Router /api/products [post]
func addProducts(c *fiber.Ctx) error {
	var products []Product

	// Проверяем, пришел ли массив или одиночный объект
	if err := c.BodyParser(&products); err != nil {
		// Попробуем спарсить как одиночный объект
		var singleProduct Product
		if err := c.BodyParser(&singleProduct); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "Invalid request"})
		}
		products = append(products, singleProduct)
	}

	query := "INSERT INTO products (name, price, description, categories) VALUES ($1, $2, $3, $4) RETURNING id"

	for i := range products {
		err := db.QueryRow(query, products[i].Name, products[i].Price, products[i].Description, pq.Array(products[i].Categories)).Scan(&products[i].ID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
		}
	}

	return c.JSON(products)
}

// @Summary Обновить данные продукта
// @Tags Products
// @Accept json
// @Produce json
// @Param id path int true "ID продукта"
// @Param product body UpdateProductRequest true "Данные продукта"
// @Success 200 {object} map[string]string "Продукт успешно обновлен"
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 500 {object} ErrorResponse "Ошибка на сервере"
// @Router /api/products/{id} [put]
func updateProduct(c *fiber.Ctx) error {
	id := c.Params("id")
	var product Product
	if err := c.BodyParser(&product); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "Invalid request"})
	}

	// Обновляем продукт с новыми категориями
	query := "UPDATE products SET name=$1, price=$2, description=$3, categories=$4 WHERE id=$5"
	_, err := db.Exec(query, product.Name, product.Price, product.Description, pq.Array(product.Categories), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Product updated successfully"})
}

// @Summary Удалить продукт
// @Tags Products
// @Accept json
// @Produce json
// @Param id path int true "ID продукта"
// @Success 200 {object} map[string]string "Продукт успешно удален"
// @Failure 500 {object} ErrorResponse "Ошибка на сервере"
// @Router /api/products/{id} [delete]
func deleteProduct(c *fiber.Ctx) error {
	id := c.Params("id")
	query := "DELETE FROM products WHERE id=$1"
	_, err := db.Exec(query, id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Product deleted successfully"})
}

func healthCheck(c *fiber.Ctx) error {
	return c.SendString("hello")
}

// @title TEST API
// @version 1.0
// @BasePath /
func main() {
	initDB()
	defer db.Close()

	app := fiber.New()

	// Настройка для отдачи статических файлов
	// `index.html` будет доступен по `/`
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendFile("./index.html")
	})

	// `admin.html` будет доступен по `/admin.html`
	app.Get("/admin", func(c *fiber.Ctx) error {
		return c.SendFile("./admin.html")
	})

	// API endpoints
	api := app.Group("/api") // Группируем API-маршруты под префиксом /api
	api.Get("/products", getProducts)
	api.Post("/products", addProducts)
	api.Put("/products/:id", updateProduct)
	api.Delete("/products/:id", deleteProduct)

	app.Get("/health", healthCheck)

	// Swagger UI будет доступен по /swagger/index.html
	app.Get("/swagger/*", swagger.HandlerDefault)

	log.Println("Сервер магазина и админ-панели запущен на порту 8080")
	log.Fatal(app.Listen(":8080"))
}
