package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type OnboardingStatus string

const (
	StatusRegistered         OnboardingStatus = "registered"
	StatusEmailVerified      OnboardingStatus = "email_verified"
	StatusCreditApproved     OnboardingStatus = "credit_approved"
	StatusCreditDenied       OnboardingStatus = "credit_denied"
	StatusOnboardingComplete OnboardingStatus = "onboarding_complete"
)

const (
	EventRegistered        = "user.registered"
	EventScoreDisqualified 	  = "user.score_disqualified"
	EventEmailVerified     = "user.email_verified"
	EventTokenVerifyFailed = "user.token_verify_failed"
	EventCreditScored      = "user.credit_scored"
	EventProfileCompleted  = "user.profile_completed"
	EventUserActivated     = "user.activated"
)

type User struct {
	ID                bson.ObjectID    `bson:"_id"                json:"id"`
	Name              string           `bson:"name"               json:"name"`
	Email             string           `bson:"email"              json:"email"`
	Bio               string           `bson:"bio"                json:"bio"`
	Status            OnboardingStatus `bson:"status"             json:"status"`
	VerificationToken string           `bson:"verification_token" json:"verification_token"`
	CreditScore       int              `bson:"credit_score"       json:"credit_score"`
	CreditApproved    bool             `bson:"credit_approved"    json:"credit_approved"`
	CreatedAt         time.Time        `bson:"created_at"         json:"created_at"`
}

// Event store payloads (minimal, BSON-encoded)
type RegisteredPayload struct {
	Name              string `bson:"name"               json:"name"`
	Email             string `bson:"email"              json:"email"`
	VerificationToken string `bson:"verification_token" json:"verification_token"`
}

type EmailVerifiedPayload struct {
	UserID string `bson:"user_id" json:"user_id"`
}

type CreditScoredPayload struct {
	UserID   string `bson:"user_id"  json:"user_id"`
	Score    int    `bson:"score"    json:"score"`
	Approved bool   `bson:"approved" json:"approved"`
}

type ProfileCompletedPayload struct {
	UserID string `bson:"user_id" json:"user_id"`
	Bio    string `bson:"bio"     json:"bio"`
}

type UserActivatedPayload struct {
	UserID string `bson:"user_id" json:"user_id"`
	Email  string `bson:"email"   json:"email"`
	Name   string `bson:"name"    json:"name"`
}

// Pure state transitions
func (u User) WithEmailVerified() User {
	u.Status = StatusEmailVerified
	return u
}

func (u User) WithCreditScore(score int, approved bool) User {
	u.CreditScore = score
	u.CreditApproved = approved
	if approved {
		u.Status = StatusCreditApproved
	} else {
		u.Status = StatusCreditDenied
	}
	return u
}

func (u User) WithProfile(bio string) User {
	u.Bio = bio
	u.Status = StatusOnboardingComplete
	return u
}
