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

const dbName = "Cluster0"
const mongoURI = "mongodb+srv://cyborg_rj:3m193mRJA123@cluster0.6biqi.mongodb.net/?retryWrites=true&w=majority"

type Aluno struct {
	ID             string `json:"id,omitempty" bson:"_id,omitempty"`
	Nome           string `json:"name" bson:"name"`
	DataNascimento string `json:"datanasc" bson:"datanasc"`
	Serie          string `json:"serie" bson:"serie"`
	Email          string `json:"email" bson:"email"`
	Idade          int    `json:"idade,omitempty" bson:"idade,omitempty"`
	Cpf            string `json:"cpf" bson:"cpf"`
}

func age(birthdate, today time.Time) int {
	today = today.In(birthdate.Location())
	ty, tm, td := today.Date()
	today = time.Date(ty, tm, td, 0, 0, 0, 0, time.UTC)
	by, bm, bd := birthdate.Date()
	birthdate = time.Date(by, bm, bd, 0, 0, 0, 0, time.UTC)
	if today.Before(birthdate) {
		return 0
	}
	age := ty - by
	anniversary := birthdate.AddDate(age, 0, 0)
	if anniversary.After(today) {
		age--
	}
	return age
}

func insereIdadeAluno(a *Aluno) {
	hoje := time.Now()
	datanasc, _ := time.Parse("02/01/2006", a.DataNascimento)
	(*a).Idade = age(hoje, datanasc)
}

func Connect() error {

	client, _ := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := client.Connect(ctx)
	db := client.Database(dbName)

	if err != nil {
		return err
	}
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

	app.Get("/aluno", func(ctx *fiber.Ctx) error {

		query := bson.D{{}}

		cursor, err := mg.Db.Collection("alunos").Find(ctx.Context(), query)
		if err != nil {
			return ctx.Status(500).SendString(err.Error())
		}

		var alunos []Aluno = make([]Aluno, 0)

		if err := cursor.All(ctx.Context(), &alunos); err != nil {
			return ctx.Status(500).SendString(err.Error())
		}

		for _, a := range alunos {
			insereIdadeAluno(&a)
		}

		return ctx.JSON(alunos)

	})

	app.Post("/aluno", func(ctx *fiber.Ctx) error {
		collection := mg.Db.Collection("alunos")

		aluno := new(Aluno)

		if err := ctx.BodyParser(aluno); err != nil {
			return ctx.Status(400).SendString(err.Error())
		}

		aluno.ID = ""

		insertionResult, err := collection.InsertOne(ctx.Context(), aluno)

		if err != nil {
			return ctx.Status(500).SendString(err.Error())
		}

		filter := bson.D{{Key: "_id", Value: insertionResult.InsertedID}}
		createdRecord := collection.FindOne(ctx.Context(), filter)

		createdAluno := &Aluno{}
		createdRecord.Decode(createdAluno)

		return ctx.Status(201).JSON(createdAluno)
	})

	app.Put("/aluno/:id", func(ctx *fiber.Ctx) error {
		idParam := ctx.Params("id")

		alunoID, err := primitive.ObjectIDFromHex(idParam)

		if err != nil {
			return ctx.SendStatus(400)
		}

		aluno := new(Aluno)

		if err := ctx.BodyParser(aluno); err != nil {
			return ctx.Status(400).SendString(err.Error())
		}

		query := bson.D{{Key: "_id", Value: alunoID}}

		update := bson.D{
			{Key: "$set",
				Value: bson.D{
					{Key: "name", Value: aluno.Nome},
					{Key: "datanasc", Value: aluno.DataNascimento},
					{Key: "serie", Value: aluno.Serie},
					{Key: "email", Value: aluno.Email},
					{Key: "cpf", Value: aluno.Cpf},
				},
			},
		}

		err = mg.Db.Collection("alunos").FindOneAndUpdate(ctx.Context(), query, update).Err()

		if err != nil {
			if err == mongo.ErrNoDocuments {
				return ctx.SendStatus(400)
			}
			return ctx.SendStatus(500)
		}

		aluno.ID = idParam

		return ctx.Status(200).JSON(aluno)
	})

	app.Delete("/aluno:id", func(ctx *fiber.Ctx) error {

		alunoID, err := primitive.ObjectIDFromHex(ctx.Params("id"))

		if err != nil {
			return ctx.SendStatus(400)
		}

		query := bson.M{"_id": alunoID}
		result, err := mg.Db.Collection("alunos").DeleteOne(ctx.Context(), &query)

		if err != nil {
			return ctx.SendStatus(500)
		}

		if result.DeletedCount < 1 {
			return ctx.SendStatus(404)
		}

		return ctx.Status(200).JSON("Aluno deletado")

	})

	log.Fatal(app.Listen(":3000"))
}
