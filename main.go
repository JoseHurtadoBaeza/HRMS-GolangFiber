package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoInstance struct {
	Client *mongo.Client
	Db     *mongo.Database
}

var mg MongoInstance

const dbName = "fiber-hrms"
const mongoURI = "mongodb://localhost:27017/" + dbName

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
			return c.Status(404).SendString(err.Error())
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

	// Update an employee in the database
	app.Put("/employee/:id", func(c *fiber.Ctx) error {

		idParam := c.Params("id")

		employeeID, err := primitive.ObjectIDFromHex(idParam)

		if err != nil {
			return c.SendStatus(404)
		}

		employee := new(Employee)

		if err := c.BodyParser(employee); err != nil {
			return c.Status(404).SendString(err.Error())
		}

		query := bson.D{{Key: "_id", Value: employeeID}}

		// Build the update query
		update := bson.D{
			{
				Key: "$set",
				Value: bson.D{
					{Key: "name", Value: employee.Name},
					{Key: "age", Value: employee.Age},
					{Key: "salary", Value: employee.Salary},
				},
			},
		}

		err = mg.Db.Collection("employees").FindOneAndUpdate(c.Context(), query, update).Err()

		if err != nil {
			if err == mongo.ErrNoDocuments {
				return c.SendStatus(404)
			}
			return c.SendStatus(500)
		}

		employee.ID = idParam

		return c.Status(200).JSON(employee)

	})

	// Delete an employee from the database
	app.Delete("/employee/:id", func(c *fiber.Ctx) error {

		employeeID, err := primitive.ObjectIDFromHex(c.Params("id"))

		if err != nil {
			return c.SendStatus(404)
		}

		query := bson.D{{Key: "_id", Value: employeeID}}

		result, err := mg.Db.Collection("employees").DeleteOne(c.Context(), &query)

		if err != nil {
			return c.SendStatus(500)
		}

		// If it didn't get deleted
		if result.DeletedCount < 1 {
			return c.SendStatus(404)
		}

		return c.Status(200).JSON("record deleted")

	})

	log.Fatal(app.Listen(":3000"))

}
