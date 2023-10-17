package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoInstance struct {
	Client *mongo.Client
	Db     *mongo.Database
}

var mg MongoInstance

const dbName = "fiber-hrms"
const mongoURI = "mongodb://localhost:27017" + dbName

type Employee struct {
	ID     string  `json:"id,omitempty" bson:"_id, omitempty"`
	Name   string  `json:"name"`
	Salary float64 `json:"salary"`
	Age    int     `json:"age"`
}

// Helps to connect Golang with MongoDB
func Connect() error {

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI)) // Using the driver to create a Client from a connection string
	if err != nil {
		return err
	}

	db := client.Database(dbName)

	mg = MongoInstance{
		Client: client,
		Db:     db,
	}

	return nil

}

func main() {

	if err := Connect(); err != nil {
		log.Fatal(err)
	}
	app := fiber.New()

	// Get list of all employees in the database
	app.Get("/employee", func(c *fiber.Ctx) error {

		query := bson.D{}

		cursor, err := mg.Db.Collection("employees").Find(c.Context(), query) // Use the collection to query the database (this method return a cursor)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		var employees []Employee = make([]Employee, 0) // Create a slice to store the returned employees in the query

		if err := cursor.All(c.Context(), &employees); err != nil { // Cursor.All will decode all of the returned elements at once
			return c.Status(500).SendString(err.Error())
		}

		return c.JSON(employees)

	})

	// Add a new employee in the database
	app.Post("/employee", func(c *fiber.Ctx) error {

		collection := mg.Db.Collection("employees")

		employee := new(Employee)

		if err := c.BodyParser(employee); err != nil {
			return c.Status(400).SendString(err.Error())
		}

		employee.ID = "" // We always want mongodb to create its own ids

		// We want to get the id from this insertionResult, because this id we'll use that to search the actual record that has just been inserted in the database
		insertionResult, err := collection.InsertOne(c.Context(), employee)

		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		// After finding that record we want to return that to the frontend
		filter := bson.D{{Key: "_id", Value: insertionResult.InsertedID}}
		createdRecord := collection.FindOne(c.Context(), filter)

		createdEmployee := &Employee{}
		createdRecord.Decode(createdEmployee)

		return c.Status(201).JSON(createdEmployee)

	})

	app.Put("/employee/:id")
	app.Delete("/employee/:id")

}
