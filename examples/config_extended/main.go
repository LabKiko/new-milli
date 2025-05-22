package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rogeecn/new-milli/config"
)

func main() {
	// 1. Set up a file source for config_extended.yaml
	fileSource := config.NewFileSource("./config_extended.yaml")
	log.Println("File source created for config_extended.yaml")

	// 2. Set up an environment variable source with a prefix
	envPrefix := "APP_EXT_"
	envSource := config.NewEnvSource(envPrefix)
	log.Printf("Environment variable source created with prefix: %s\n", envPrefix)

	// 3. Create a composite config source (environment overrides file)
	// The order matters: sources listed later override earlier ones.
	compositeSource := config.NewCompositeSource(fileSource, envSource)
	log.Println("Composite source created (ENV overrides File)")

	// 4. Create a new config instance with the composite source
	cfg := config.NewConfig(compositeSource)
	log.Println("Config instance created")

	// 5. Load the initial configuration
	if err := cfg.Load(); err != nil {
		log.Fatalf("Error loading initial configuration: %v", err)
	}
	log.Println("Initial configuration loaded successfully")

	// 6. Print initial values
	fmt.Println("\n--- Initial Configuration Values ---")

	appName, err := cfg.GetString("application.name")
	if err != nil {
		log.Printf("Error getting application.name: %v. Using default: 'Default App Name'", err)
		appName = "Default App Name"
	}
	fmt.Printf("Application Name: %s (string)\n", appName)

	appPort, err := cfg.GetInt("application.port")
	if err != nil {
		log.Printf("Error getting application.port: %v. Using default: 8000", err)
		appPort = 8000
	}
	fmt.Printf("Application Port: %d (int)\n", appPort)

	appDebug, err := cfg.GetBool("application.debug_mode")
	if err != nil {
		log.Printf("Error getting application.debug_mode: %v. Using default: false", err)
		appDebug = false
	}
	fmt.Printf("Application Debug Mode: %t (bool)\n", appDebug)

	appFeatures, err := cfg.GetStringSlice("application.features")
	if err != nil {
		log.Printf("Error getting application.features: %v. Using default: []", err)
		appFeatures = []string{}
	}
	fmt.Printf("Application Features: %v (string slice)\n", appFeatures)

	retryAttempts, err := cfg.GetInt("application.settings.retry_attempts")
	if err != nil {
		log.Printf("Error getting application.settings.retry_attempts: %v. Using default: 1", err)
		retryAttempts = 1
	}
	fmt.Printf("Application Settings - Retry Attempts: %d (int)\n", retryAttempts)

	appSettings, err := cfg.Get("application.settings")
	if err != nil {
		log.Printf("Error getting application.settings: %v. Using default: map[]", err)
		appSettings = make(map[string]interface{})
	}
	fmt.Printf("Application Settings (raw map): %v (map)\n", appSettings)

	// Example of accessing a non-existent key
	dbHost, err := cfg.GetString("database.host")
	if err != nil {
		if errors.Is(err, config.ErrNotFound) {
			fmt.Printf("Non-existent key (database.host with default): default_db_host (Reason: %v)\n", err)
			dbHost = "default_db_host" // Assign default
		} else {
			fmt.Printf("Error getting database.host: %v\n", err)
			dbHost = "error_db_host" // Indicate error
		}
	} else {
		fmt.Printf("Database Host: %s\n", dbHost)
	}
	fmt.Println("--- End of Initial Configuration ---")

	// 7. Explain how to set environment variables to override
	fmt.Println("\n--- How to Override with Environment Variables ---")
	fmt.Printf("To override 'application.name', set: %sAPPLICATION_NAME=\"My App From ENV\"\n", envPrefix)
	fmt.Printf("To override 'application.port', set: %sAPPLICATION_PORT=9090\n", envPrefix)
	fmt.Printf("To override 'application.debug_mode', set: %sAPPLICATION_DEBUG_MODE=false\n", envPrefix)
	fmt.Printf("To override 'application.features', set: %sAPPLICATION_FEATURES='[\\\"featC\\\",\\\"featD\\\"]' (note JSON array format for slices, or comma-separated for simple env vars if parser supports it)\n", envPrefix)
	fmt.Printf("To override 'application.settings.retry_attempts', set: %sAPPLICATION_SETTINGS_RETRY_ATTEMPTS=5\n", envPrefix)
	fmt.Println("Shell examples:")
	fmt.Printf("  export %sAPPLICATION_NAME=\"My App From ENV\"\n", envPrefix)
	fmt.Printf("  export %sAPPLICATION_PORT=9090\n", envPrefix)
	fmt.Println("Restart the program after setting environment variables to see their effect on initial load.")
	fmt.Println("--- End of Override Explanation ---")

	// 8. Set up a watcher for configuration changes
	watchCh, err := cfg.Watch()
	if err != nil {
		log.Fatalf("Error setting up config watcher: %v", err)
	}
	if watchCh == nil {
		log.Println("Watcher channel is nil. Live reload might not be supported by all sources (e.g., basic EnvSource).")
	} else {
		log.Println("\nConfiguration watcher started. Monitoring for changes in sources that support watching (e.g., config_extended.yaml)...")
	}


	// 9. Goroutine to listen for changes
	if watchCh != nil {
		go func() {
			for {
				_, ok := <-watchCh
				if !ok {
					log.Println("Watch channel closed. Stopping watch goroutine.")
					return
				}

				log.Println("Configuration change detected!")
				// Reload the configuration
				if err := cfg.Load(); err != nil {
					log.Printf("Error reloading configuration: %v", err)
					continue
				}
				log.Println("Configuration reloaded successfully.")

				// Print some values to show updates
				fmt.Println("\n--- Updated Configuration Values ---")
				updatedAppName, _ := cfg.GetString("application.name")
				fmt.Printf("Application Name: %s\n", updatedAppName)

				updatedAppPort, _ := cfg.GetInt("application.port")
				fmt.Printf("Application Port: %d\n", updatedAppPort)

				updatedAppDebug, _ := cfg.GetBool("application.debug_mode")
				fmt.Printf("Application Debug Mode: %t\n", updatedAppDebug)

				updatedAppFeatures, _ := cfg.GetStringSlice("application.features")
				fmt.Printf("Application Features: %v\n", updatedAppFeatures)

				updatedRetryAttempts, _ := cfg.GetInt("application.settings.retry_attempts")
				fmt.Printf("Application Settings - Retry Attempts: %d\n", updatedRetryAttempts)
				fmt.Println("--- End of Updated Configuration ---")
			}
		}()
	}

	// 10. Simulate a change by informing the user
	fmt.Println("\n--- Live Configuration Reload Test ---")
	fmt.Println("The program is now watching for changes in 'examples/config_extended/config_extended.yaml'.")
	fmt.Println("To test live reload:")
	fmt.Println("1. Open 'examples/config_extended/config_extended.yaml' in a text editor.")
	fmt.Println("2. Modify a value (e.g., change `application.name` to `\"My App Updated Via File\"` or `application.port` to `8081`).")
	fmt.Println("3. Save the file.")
	fmt.Println("The changes should be detected and printed above (if watching is active).")
	fmt.Println("---")

	// 11. Keep the program running or wait for a signal
	fmt.Println("\nProgram will run for 60 seconds to allow for manual file changes, or press Ctrl+C to exit.")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-time.After(60 * time.Second):
		log.Println("60 seconds elapsed. Exiting program.")
	case s := <-sigChan:
		log.Printf("Received signal %s. Exiting program.", s)
	}

	if err := cfg.Close(); err != nil {
		log.Printf("Error closing config: %v", err)
	}
	log.Println("Program exited.")
}
