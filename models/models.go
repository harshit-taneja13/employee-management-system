package models

import "go.mongodb.org/mongo-driver/v2/bson"


type Employee struct {
	ID bson.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Name string `json:"name,omitempty" bson:"name,omitempty"`
	Department string `json:"department,omitempty" bson:"department,omitempty"`
	Age int `json:"age,omitempty" bson:"age,omitempty"`
	Email string `json:"email,omitempty" bson:"email,omitempty"`
}
