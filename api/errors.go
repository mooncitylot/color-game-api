package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
)

// Helper function to get caller information
func getCallerInfo() string {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return "[unknown]"
	}
	return fmt.Sprintf("[%s:%d]", filepath.Base(file), line)
}

type HandlerError struct {
	ErrorName        string `json:"errorName"`
	Description      string `json:"description"`
	PossibleSolution string `json:"possibleSolution"`
	CallerInfo       string `json:"callerInfo"`
}

var ErrGET = fmt.Errorf("GET method required for this endpoint")
var ErrPOST = fmt.Errorf("POST method required for this endpoint")
var ErrPUT = fmt.Errorf("PUT method required for this endpoint")
var ErrInvalidPrivelege = fmt.Errorf("invalid authentication privileges")

func (app *Application) invalidCredentials(w http.ResponseWriter, r *http.Request, err error) {
	w.WriteHeader(http.StatusUnauthorized)
	errAuthorizingUser := HandlerError{
		ErrorName:        "Error Authorizing User",
		Description:      err.Error(),
		PossibleSolution: "Retry with proper credentials",
		CallerInfo:       getCallerInfo(),
	}
	json.NewEncoder(w).Encode(errAuthorizingUser)
}

func (app *Application) invalidAuthorization(w http.ResponseWriter, r *http.Request, err error) {
	w.WriteHeader(http.StatusUnauthorized)
	errAuthorizingEndpoint := HandlerError{
		ErrorName:        "Error Authenticating for Endpoint",
		Description:      "Invalid Authentication",
		PossibleSolution: "Check your headers and ensure you're submitting a valid token",
		CallerInfo:       getCallerInfo(),
	}
	json.NewEncoder(w).Encode(errAuthorizingEndpoint)
}

func (app *Application) requirePostMethod(w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("Allow", http.MethodPost)
	w.WriteHeader(http.StatusMethodNotAllowed)
	postMethodRequired := HandlerError{
		ErrorName:        "Post Method Required",
		Description:      err.Error() + " you used: " + r.Method,
		PossibleSolution: "Use POST method",
		CallerInfo:       getCallerInfo(),
	}
	json.NewEncoder(w).Encode(postMethodRequired)
}

func (app *Application) requirePutMethod(w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("Allow", http.MethodPut)
	w.WriteHeader(http.StatusMethodNotAllowed)
	postMethodRequired := HandlerError{
		ErrorName:        "PUT Method Required",
		Description:      err.Error(),
		PossibleSolution: "Use PUT method",
		CallerInfo:       getCallerInfo(),
	}
	json.NewEncoder(w).Encode(postMethodRequired)
}

func (app *Application) badJSONRequest(w http.ResponseWriter, r *http.Request, err error) {
	w.WriteHeader(http.StatusBadRequest)
	jsonErr := HandlerError{
		ErrorName:        "Error Parsing JSON",
		Description:      err.Error(),
		PossibleSolution: "Double check your JSON formatting",
		CallerInfo:       getCallerInfo(),
	}
	json.NewEncoder(w).Encode(jsonErr)
}

func (app *Application) internalServerError(w http.ResponseWriter, r *http.Request, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	errorStoringSessionToken := HandlerError{
		ErrorName:        "Internal Server Error",
		Description:      err.Error(),
		PossibleSolution: "Internal Server Error requiring support",
		CallerInfo:       getCallerInfo(),
	}
	json.NewEncoder(w).Encode(errorStoringSessionToken)
}

func (app *Application) userAlreadyExists(w http.ResponseWriter, r *http.Request, err error) {
	userExists := HandlerError{
		ErrorName:        "User Exists",
		Description:      "There is already a user with this email address",
		PossibleSolution: "Advise user to login with their credentials",
		CallerInfo:       getCallerInfo(),
	}
	w.WriteHeader(http.StatusConflict)
	json.NewEncoder(w).Encode(userExists)
}

func (app *Application) badRequest(w http.ResponseWriter, r *http.Request, err error) {
	badRequest := HandlerError{
		ErrorName:        "Bad Request",
		Description:      err.Error(),
		PossibleSolution: "Check your request parameters",
		CallerInfo:       getCallerInfo(),
	}
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(badRequest)
}
