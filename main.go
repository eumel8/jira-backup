package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"

	//"net/http/httputil"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	BaseURL       string `json:"baseurl"`
	SpaceKey      string `json:"spacekey"`
	Token         string `json:"token"`
	BackupDir     string `json:"backupdir"`
	Timeout       int    `json:"timeout"`
	S3Bucket      string `json:"s3bucket"`
	S3KeyPrefix   string `json:"s3keyprefix"`
	S3Region      string `json:"s3region"`
	S3AccessKey   string `json:"s3accesskey"`
	S3SecretKey   string `json:"s3secretkey"`
	RetentionDays int  `json:"retentiondays"`
}

func loadConfigFromFile(path string) (Config, error) {

	// File does not exist, return 0
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return Config{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var config Config
	err = json.Unmarshal(data, &config)
	return config, err
}

// overrideWithEnv overrides the config with environment variables if they are set
func overrideWithEnv(config *Config) {

	if baseURL := os.Getenv("JIRA_BASE_URL"); baseURL != "" {
		config.BaseURL = baseURL
	}

	if spaceKey := os.Getenv("JIRA_SPACE_KEY"); spaceKey != "" {
		config.SpaceKey = spaceKey
	}

	if token := os.Getenv("JIRA_TOKEN"); token != "" {
		config.Token = token
	}

	if backupDir := os.Getenv("JIRA_BACKUP_DIR"); backupDir != "" {
		config.BackupDir = backupDir
	}

	if s3Bucket := os.Getenv("JIRA_S3_BUCKET"); s3Bucket != "" {
		config.S3Bucket = s3Bucket
	}

	if s3Region := os.Getenv("JIRA_S3_REGION"); s3Region != "" {
		config.S3Region = s3Region
	}

	if s3KeyPrefix := os.Getenv("JIRA_S3_KEY_PREFIX"); s3KeyPrefix != "" {
		config.S3KeyPrefix = s3KeyPrefix
	}

	if s3AccessKey := os.Getenv("JIRA_S3_ACCESS_KEY"); s3AccessKey != "" {
		config.S3AccessKey = s3AccessKey
	}

	if s3SecretKey := os.Getenv("JIRA_S3_SECRET_KEY"); s3SecretKey != "" {
		config.S3SecretKey = s3SecretKey
	}

	if timeoutStr := os.Getenv("JIRA_TIMEOUT"); timeoutStr != "" {
		if timeout, err := strconv.Atoi(timeoutStr); err == nil {
			config.Timeout = timeout
		}
	}

	if retentionDaysStr := os.Getenv("JIRA_RENTENTION_DAYS"); retentionDaysStr != "" {
		if retentionDays, err := strconv.Atoi(retentionDaysStr); err == nil {
			config.RetentionDays = retentionDays
		}
	}
}

// triggerBackup starts a backup job for the specified space
func triggerBackup(cfg Config, client *http.Client) (int, error) {
	url := fmt.Sprintf("%s/rest/api/backup-restore/backup/space", cfg.BaseURL)
	bearer := fmt.Sprintf("ksso-token %s", cfg.Token)
	payload := map[string]interface{}{
		"spaceKeys":       []string{cfg.SpaceKey},
		"keepPermanently": false,
		"fileNamePrefix":  "",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearer)

	res, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		detail, _ := io.ReadAll(res.Body)
		return 0, fmt.Errorf("unexpected status: %d - %s", res.StatusCode, string(detail))
	}

	var resp struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return 0, err
	}

	return resp.ID, nil
}

// pollJob checks the status of the backup job until it is finished or failed
func pollJob(cfg Config, client *http.Client, jobID int) (string, error) {
	url := fmt.Sprintf("%s/rest/api/backup-restore/jobs/%d", cfg.BaseURL, jobID)
	for {
		bearer := fmt.Sprintf("ksso-token %s", cfg.Token)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", bearer)

		res, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer res.Body.Close()

		var resp struct {
			JobState   string `json:"jobState"`
			FileName   string `json:"fileName"`
			FileExists bool   `json:"fileExists"`
		}

		if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
			return "", err
		}
		log.Println("⌛ Status ", resp.JobState)
		switch resp.JobState {
		case "FINISHED":
			return resp.FileName, nil
		case "FAILED":
			return "", fmt.Errorf("backup job failed")
		default:
			log.Println("🔄 Backup in progress...")
			time.Sleep(5 * time.Second)
		}
	}
}

// download the backup file and store them to tmp directory
func downloadBackupFile(cfg Config, client *http.Client, downloadURL string, downloadFile string) (string, error) {

	if cfg.Timeout == 0 {
		cfg.Timeout = 10
	}

	fullURL := cfg.BaseURL + downloadURL
	req, _ := http.NewRequest("GET", fullURL, nil)
	disposition := fmt.Sprintf("attachment; filename=%s", downloadFile)
	bearer := fmt.Sprintf("ksso-token %s", cfg.Token)
	req.Header.Set("Authorization", bearer)
	req.Header.Set("content-disposition", disposition)

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	filePath := filepath.Join(cfg.BackupDir, downloadFile)
	bckFile, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer bckFile.Close()

	_, err = io.Copy(bckFile, res.Body)
	if err != nil {
		return "", err
	}

	return bckFile.Name(), nil
}

// TODO S3 upload
func uploadToS3(cfg Config, filePath string) error {

	return nil
	//sess := session.Must(session.NewSession(&aws.Config{
	//	Region: aws.String(cfg.S3Region),
	//}))
	//uploader := s3manager.NewUploader(sess)

	//file, err := os.Open(filePath)
	//if err != nil {
	//	return err
	//}
	//defer file.Close()

	//key := fmt.Sprintf("%s/%s", cfg.S3KeyPrefix, filepath.Base(filePath))
	//_, err = uploader.Upload(&s3manager.UploadInput{
	//	Bucket: aws.String(cfg.S3Bucket),
	//	Key:    aws.String(key),
	//	Body:   file,
	//})

	//return err
}

func cleanupOldBackups(backupDir, spaceKey string, retentionDays int) error {
    if retentionDays <= 0 {
        log.Println("Retention disabled, skipping cleanup.")
        return nil
    }

    files, err := os.ReadDir(backupDir)
    if err != nil {
        return fmt.Errorf("cannot read backup dir: %w", err)
    }

    cutoff := time.Now().AddDate(0, 0, -retentionDays)
    prefix := fmt.Sprintf("Confluence-space-export-%s-", spaceKey)
    suffix := ".zip"

    for _, f := range files {
        if f.IsDir() {
            continue
        }

        name := f.Name()
		log.Printf("Check backup file: %s", name)
        if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) {
            // not a backup file, skip
            continue
        }

        fullPath := filepath.Join(backupDir, name)
        info, err := os.Stat(fullPath)
        if err != nil {
            log.Printf("Skipping file (stat error): %s", name)
            continue
        }

        if info.ModTime().Before(cutoff) {
            log.Printf("Deleting old backup: %s", name)
            if err := os.Remove(fullPath); err != nil {
                log.Printf("Failed to delete %s: %v", name, err)
            }
        }
    }
    return nil
}


func main() {

	cfgFile := "config.json"
	if envCfg := os.Getenv("JIRA_CONFIG"); envCfg != "" {
		cfgFile = envCfg
	}

	// try load config from json file
	cfg, err := loadConfigFromFile(cfgFile)
	if err != nil {
		// config file is optional, only log info
		log.Printf("❓Error loading config from file: %v", err)
	}

	// lookup config vars
	overrideWithEnv(&cfg)
	if cfg.BaseURL == "" || cfg.SpaceKey == "" || cfg.Token == "" {
		log.Fatal("❌ Missing required configuration: BaseURL, SpaceKey, or Token must be set")
	}

	// Initialize the HTTP client with the correct timeout after config is loaded
	client := &http.Client{
		Timeout: time.Duration(cfg.Timeout) * time.Minute,
		Transport: &http.Transport{
			// use system proxy settings
			Proxy: http.ProxyFromEnvironment,
			// Disable HTTP/2 to avoid issues with some Jira instances
			ForceAttemptHTTP2: false,
			// Disable TLS verification for self-signed certificates (not recommended for production)
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	log.Println("▶️ Triggering backup...")
	jobID, err := triggerBackup(cfg, client)
	if err != nil {
		log.Fatalf("❌ Failed to start backup: %v", err)
	}

	log.Printf("⏩ Backup job started: %d", jobID)
	downloadFile, err := pollJob(cfg, client, jobID)
	if err != nil {
		log.Fatalf("❌ Polling failed: %v", err)
	}

	downloadURL := fmt.Sprintf("/rest/api/backup-restore/jobs/%d/download", jobID)

	log.Println("⬇️ Downloading backup file...")
	localPath, err := downloadBackupFile(cfg, client, downloadURL, downloadFile)
	if err != nil {
		log.Fatalf("❌ Download failed: %v", err)
	}

	log.Println("🧹 Cleanup old backups")
	if err := cleanupOldBackups(cfg.BackupDir, cfg.SpaceKey, cfg.RetentionDays); err != nil {
   		log.Printf("Cleanup warning: %v", err)
	}

	defer os.Remove(localPath)

	log.Println("🔄 Uploading to S3 (todo)... ", localPath)
	log.Println("✅ Backup complete.")
	os.Exit(0) // Exit early for testing purposes
	if err := uploadToS3(cfg, localPath); err != nil {
		log.Fatalf("❌ S3 upload failed: %v", err)
	}

}
