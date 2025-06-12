package entity

type UserAuth struct {
	ID             string `json:"id"`
	HashedPassword string `json:"hashed_password"`
}
