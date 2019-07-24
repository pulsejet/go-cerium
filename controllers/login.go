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

	u "github.com/pulsejet/go-cerium/utils"
	"go.mongodb.org/mongo-driver/bson"
)

// AuthCode : code and redirect uri to POST
type AuthCode struct {
	Code        string `json:"code"`
	RedirectURI string `json:"redirect_uri"`
}

// AccessTokenResponse : access token received from SSO
type AccessTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// ProfileResponse : Profile as received from SSO
type ProfileResponse struct {
	ID             int    `json:"id"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	RollNumber     string `json:"roll_number"`
	ProfilePicture string `json:"profile_picture"`
	Email          string `json:"email"`
}

// Claims : JWT claims stored on client side
type Claims struct {
	RollNumber string `json:"roll_number"`
	jwt.StandardClaims
}

var jwtKey = []byte(os.Getenv("JWT_KEY"))

// Login : API handler for logging in with SSO auth code
var Login = func(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// Get JWT
		rno := GetRollNo(w, r, true)
		if rno == "" {
			return
		}

		// Get profile
		collection := u.Collection(r.Context(), "users")
		user := &ProfileResponse{}
		err := collection.FindOne(r.Context(), bson.M{"rollnumber": rno}).Decode(user)
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
	authToken := os.Getenv("AUTH_TOKEN")
	authTemplate := "code=%s&redirect_uri=%s&grant_type=authorization_code"

	var jsonStr = []byte(fmt.Sprintf(authTemplate, code.Code, code.RedirectURI))
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", authToken))
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
	accessTokenResponse := &AccessTokenResponse{}
	err = json.NewDecoder(resp.Body).Decode(accessTokenResponse)

	// Error if we didn't get the access token
	if err != nil {
		log.Println(err)
		u.Respond(w, u.Message(false, err.Error()), 500)
		return
	}

	profileURL := os.Getenv("OAUTH_PROFILE")

	// Make request for profile
	req, err = http.NewRequest("GET", profileURL, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessTokenResponse.AccessToken))
	resp, err = client.Do(req)
	if err != nil {
		log.Println(err)
		u.Respond(w, u.Message(false, err.Error()), 500)
		return
	}
	defer resp.Body.Close()

	// Read profile
	profileResponse := &ProfileResponse{}
	err = json.NewDecoder(resp.Body).Decode(profileResponse)
	if err != nil {
		log.Println(err)
		u.Respond(w, u.Message(false, err.Error()), 500)
		return
	}

	// Save Profile
	rno := profileResponse.RollNumber
	collection := u.Collection(r.Context(), "users")

	// Allow creation
	var ropts options.ReplaceOptions
	ropts.SetUpsert(true)

	// Upsert profile
	_, err = collection.ReplaceOne(
		r.Context(), bson.M{"rollnumber": rno}, profileResponse, &ropts)

	// Set cookie
	SetCookie(w, rno)

	// Return profile
	u.Respond(w, profileResponse, 200)
}

// GetRollNo : helper function to get claimed roll number from JWT
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

// Logout : API handler for logging out
var Logout = func(w http.ResponseWriter, r *http.Request) {
	// Finally, we set the client cookie for "token" as the JWT we just generated
	http.SetCookie(w, &http.Cookie{
		Name:    "token",
		Value:   "",
		Expires: time.Unix(0, 0),
	})

	// Return profile
	u.Respond(w, "", 204)
}

// SetCookie : helper function to set JWT cookie
var SetCookie = func(w http.ResponseWriter, rno string) {
	// Declare the expiration time of the token
	expirationTime := time.Now().Add(24 * time.Hour)

	// Create the JWT claims
	claims := &Claims{
		RollNumber: rno,
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
}
