package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/renniemaharaj/thewriterco-auth-go/pkg/transformer/gemi"
)

// API keys pool
var availableKeysChan chan gemi.APIKey
var once sync.Once // ensures initialization happens only once

// Initialize API key pool
func InitAPIKeys(keys []gemi.APIKey) {
	once.Do(func() {
		availableKeysChan = make(chan gemi.APIKey, len(keys))
		for _, key := range keys {
			availableKeysChan <- key
		}
		log.Printf("API Key Pool Initialized with %d keys", len(keys))
	})
}

// HydrateAPIPool loads API keys from an environment variable.
func HydrateAPIPool(envVar string) ([]gemi.APIKey, error) {
	jsonStr := os.Getenv(envVar)
	if jsonStr == "" {
		return nil, fmt.Errorf("environment variable %s is empty", envVar)
	}

	var keys []gemi.APIKey
	if err := json.Unmarshal([]byte(jsonStr), &keys); err != nil {
		return nil, fmt.Errorf("failed to unmarshal API keys: %w", err)
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("no API keys found in environment variable %s", envVar)
	}

	log.Printf("Loaded %d API keys from environment variable %s", len(keys), envVar)

	return keys, nil
}

func WaitFreeSession(ctx context.Context) (*gemi.Session, func(), error) {
	log.Println("Waiting for available key...")

	// Non-blocking key retrieval
	select {
	case api := <-availableKeysChan:
		log.Printf("Using key: %s", api.Key)

		log.Println("Creating model...")
		model, cleanup, err := gemi.Model(ctx, api.Key, api.Base)
		if err != nil {
			availableKeysChan <- api                      // return key if model creation fails
			log.Printf("Freeing key (Fail): %s", api.Key) // Log key release
			return nil, nil, fmt.Errorf("error creating model: %w", err)
		}

		// Custom cleanup to return the key back when done
		cleanupFunc := func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Recovered from panic in cleanup: %v", r)
				}
				log.Printf("Freeing key (Finish): %s", api.Key) // Log key release
				availableKeysChan <- api                        // Always return key
			}()

			cleanup() // Call original cleanup
		}

		session := gemi.Session{Model: model}

		return &session, cleanupFunc, nil

	default:
		return nil, nil, fmt.Errorf("no API keys available")
	}
}
