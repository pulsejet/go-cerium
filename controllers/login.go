package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
	"go.mongodb.org/mongo-driver/mongo/options"

	u "github.com/pulsejet/cerium/utils"
	"go.mongodb.org/mongo-driver/bson"
)

type AuthCode struct {
	Code        string `json:"code"`
	RedirectUri string `json:"redirect_uri"`
}

type AccessTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

type ProfileResponse struct {
	Id             int    `json:"id"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	RollNumber     string `json:"roll_number"`
	ProfilePicture string `json:"profile_picture"`
	Email          string `json:"email"`
}

type Claims struct {
	RollNumber string `json:"roll_number"`
	jwt.StandardClaims
}

var jwtKey = []byte(os.Getenv("JWT_KEY"))

var Login = func(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// Get JWT
		rno := GetRollNo(w, r, true)
		if rno == "" {
			return
		}

		// Get profile
		collection := u.Collection("users")
		user := &ProfileResponse{}
		err := collection.FindOne(u.Context(), bson.M{"rollnumber": rno}).Decode(user)
		if err != nil {
			u.Respond(w, u.Message(false, err.Error()), 500)
			return
		}

		// Return profile
		u.Respond(w, user, 200)
		return
	}

	// Get the auth code
	code := &AuthCode{}
	err := json.NewDecoder(r.Body).Decode(code)
	if err != nil {
		u.Respond(w, u.Message(false, err.Error()), 400)
		return
	}

	// Make request for access token
	url := os.Getenv("OAUTH_URL")
	AUTH_TOKEN := os.Getenv("AUTH_TOKEN")
	AUTH_TEMPLATE := "code=%s&redirect_uri=%s&grant_type=authorization_code"

	var jsonStr = []byte(fmt.Sprintf(AUTH_TEMPLATE, code.Code, code.RedirectUri))
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", AUTH_TOKEN))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")

	// Fire request for access token
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		u.Respond(w, u.Message(false, err.Error()), 401)
		return
	}
	defer resp.Body.Close()

	// Ensure we got the access token
	if resp.StatusCode != 200 {
		u.Respond(w, u.Message(false, "Error"), resp.StatusCode)
		return
	}

	// Read the access token
	access_token_response := &AccessTokenResponse{}
	err = json.NewDecoder(resp.Body).Decode(access_token_response)

	// Error if we didn't get the access token
	if err != nil {
		log.Println(err)
		u.Respond(w, u.Message(false, err.Error()), 500)
		return
	}

	PROFILE_URL := os.Getenv("OAUTH_PROFILE")

	// Make request for profile
	req, err = http.NewRequest("GET", PROFILE_URL, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", access_token_response.AccessToken))
	resp, err = client.Do(req)
	if err != nil {
		log.Println(err)
		u.Respond(w, u.Message(false, err.Error()), 500)
		return
	}
	defer resp.Body.Close()

	// Read profile
	profile_response := &ProfileResponse{}
	err = json.NewDecoder(resp.Body).Decode(profile_response)
	if err != nil {
		log.Println(err)
		u.Respond(w, u.Message(false, err.Error()), 500)
		return
	}

	// Save Profile
	rno := profile_response.RollNumber
	collection := u.Collection("users")

	// Allow creation
	var ropts options.ReplaceOptions
	ropts.SetUpsert(true)

	// Upsert profile
	_, err = collection.ReplaceOne(
		u.Context(), bson.M{"rollnumber": rno}, profile_response, &ropts)

	// Declare the expiration time of the token
	expirationTime := time.Now().Add(24 * time.Hour)

	// Create the JWT claims
	claims := &Claims{
		RollNumber: profile_response.RollNumber,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	// Declare the token with the algorithm used for signing, and the claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Create the JWT string
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Finally, we set the client cookie for "token" as the JWT we just generated
	http.SetCookie(w, &http.Cookie{
		Name:    "token",
		Value:   tokenString,
		Expires: expirationTime,
	})

	// Return profile
	u.Respond(w, profile_response, 200)
}

var GetRollNo = func(w http.ResponseWriter, r *http.Request, throw bool) string {
	c, err := r.Cookie("token")
	if err != nil {
		if err == http.ErrNoCookie {
			// If the cookie is not set, return an unauthorized status
			if throw {
				w.WriteHeader(http.StatusUnauthorized)
			}
			return ""
		}
		// For any other type of error, return a bad request status
		if throw {
			w.WriteHeader(http.StatusBadRequest)
		}
		return ""
	}

	// Get the JWT string from the cookie
	tknStr := c.Value

	// Initialize a new instance of `Claims`
	claims := &Claims{}

	// Parse claims
	tkn, err := jwt.ParseWithClaims(tknStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	// Check invalid claims
	if !tkn.Valid {
		if throw {
			w.WriteHeader(http.StatusUnauthorized)
		}
		return ""
	}

	// Check invalid signatures and other errors
	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			if throw {
				w.WriteHeader(http.StatusUnauthorized)
			}
			return ""
		}
		if throw {
			w.WriteHeader(http.StatusBadRequest)
		}
		return ""
	}

	return claims.RollNumber
}
