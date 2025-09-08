package mux

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Respond sends an http json encoded response to the client.
func Respond(ctx context.Context, w http.ResponseWriter, statusCode int, data any) error {
	//set the statusCode for logging
	if err := setStatusCode(ctx, statusCode); err != nil {
		return fmt.Errorf("failed to set the statusCode to metatadat: %w", err)
	}

	//to catch both request cancelling and deadline on the request
	if err := ctx.Err(); err != nil {
		return errors.New("client is disconnected")
	}

	if statusCode == http.StatusNoContent {
		w.WriteHeader(statusCode)
		return nil
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		return fmt.Errorf("encoding response: %w", err)
	}

	return nil
}
