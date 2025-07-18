package database

import (
	"context"
	"errors"
	"fmt"
	bussinesslogic "onlineChat/bussinessLogic"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDB struct {
	Mongo *mongo.Database
}

func NewMongoDB(uri, dbName string) (MongoDB, error) {
	// ساخت client options
	clientOpts := options.Client().ApplyURI(uri)

	// ساخت context با timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// اتصال به دیتابیس
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return MongoDB{}, err
	}

	// انتخاب دیتابیس
	db := client.Database(dbName)

	return MongoDB{Mongo: db}, nil
}

func (m MongoDB) InsertUser(name, pass, phoneNumber string) error {
	type Data struct {
		Username    string `bson:"username"`
		Password    string `bson:"password"`
		PhoneNumber string `bson:"phoneNumber"`
	}
	var data Data
	data.Username = name
	data.Password = pass
	data.PhoneNumber = phoneNumber

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	count, err := m.Mongo.Collection("users").CountDocuments(ctx, bson.M{"username": name /*"phoneNumber": phoneNumber*/})
	fmt.Println("is: ", count)
	if err != nil || count > 0 {
		fmt.Println("reading error: ", err)

		return err
	}

	_, err = m.Mongo.Collection("users").InsertOne(ctx, data)
	if err != nil {
		panic(err)
	}
	return nil
}
func (m MongoDB) CheackUserById(name, pass string) (any, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	type Data struct {
		ID          primitive.ObjectID `bson:"_id"`
		Username    string             `bson:"username"`
		Password    string             `bson:"password"`
		PhoneNumber string             `bson:"phoneNumber"`
	}
	var data Data
	err := m.Mongo.Collection("users").FindOne(ctx, bson.M{"username": name}).Decode(&data)
	if err != nil {
		return "", "", err
	}
	if pass != data.Password {
		return "", "", errors.New("password incorrect")

	}
	return data.ID, data.PhoneNumber, nil
}
func (m MongoDB) InsertMessagesIntoDatabase(username, msg string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	type Doc struct {
		Username string    `bson:"username"`
		Message  string    `bson:"message"`
		Date     time.Time `bson:"time"`
	}
	doc := Doc{
		Username: username,
		Message:  msg,
		Date:     time.Now(),
	}
	_, err := m.Mongo.Collection("messages").InsertOne(ctx, doc)
	if err != nil {
		panic(err)
	}
	return nil
}

// type ChatMessages struct {
// 	Username string    `bson:"username"`
// 	Message  string    `bson:"message"`
// 	Date     time.Time `bson:"time"`
// }

func (m MongoDB) ReadAllMessagesFromDatabase() ([]bussinesslogic.ChatMessages, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cursor, err := m.Mongo.Collection("messages").Find(ctx, bson.M{})
	if err != nil {
		return []bussinesslogic.ChatMessages{}, err
	}
	defer cursor.Close(ctx)

	var messages []bussinesslogic.ChatMessages
	err = cursor.All(ctx, &messages)
	if err != nil {
		return []bussinesslogic.ChatMessages{}, err
	}
	fmt.Println("messages: ", messages)
	return messages, nil

}
