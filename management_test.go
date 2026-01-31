package tryl

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewManagementClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		sessionToken string
		wantErr     bool
	}{
		{
			name:         "valid session token",
			sessionToken: "session_token_12345",
			wantErr:      false,
		},
		{
			name:         "empty session token",
			sessionToken: "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, err := NewManagementClient(tt.sessionToken)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewManagementClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewManagementClient() returned nil client")
			}
		})
	}
}

func TestClient_ListProjects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		response   ProjectList
		wantErr    bool
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			response: ProjectList{
				Projects: []Project{
					{
						ID:          "proj_test123",
						Name:        "Test Project",
						Environment: "test",
						CreatedAt:   time.Now(),
						UpdatedAt:   time.Now(),
					},
				},
			},
			wantErr: false,
		},
		{
			name:       "empty list",
			statusCode: http.StatusOK,
			response: ProjectList{
				Projects: []Project{},
			},
			wantErr: false,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/projects" {
					t.Errorf("expected path /v1/projects, got %s", r.URL.Path)
				}
				if r.Method != "GET" {
					t.Errorf("expected GET method, got %s", r.Method)
				}

				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					json.NewEncoder(w).Encode(tt.response)
				}
			}))
			defer server.Close()

			client, _ := NewManagementClient("session_token", WithBaseURL(server.URL))
			result, err := client.ListProjects(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("ListProjects() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(result.Projects) != len(tt.response.Projects) {
					t.Errorf("ListProjects() returned %d projects, want %d",
						len(result.Projects), len(tt.response.Projects))
				}
			}
		})
	}
}

func TestClient_CreateProject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		request    CreateProjectRequest
		statusCode int
		response   CreateProjectResponse
		wantErr    bool
	}{
		{
			name: "success",
			request: CreateProjectRequest{
				Name:        "New Project",
				Environment: "test",
			},
			statusCode: http.StatusCreated,
			response: CreateProjectResponse{
				Project: Project{
					ID:          "proj_new123",
					Name:        "New Project",
					Environment: "test",
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				},
				APIKey: "actlog_test_1234567890abcdef1234567890abcdef",
			},
			wantErr: false,
		},
		{
			name: "validation error",
			request: CreateProjectRequest{
				Name:        "",
				Environment: "test",
			},
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/projects" {
					t.Errorf("expected path /v1/projects, got %s", r.URL.Path)
				}
				if r.Method != "POST" {
					t.Errorf("expected POST method, got %s", r.Method)
				}

				var req CreateProjectRequest
				json.NewDecoder(r.Body).Decode(&req)

				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusCreated {
					json.NewEncoder(w).Encode(tt.response)
				}
			}))
			defer server.Close()

			client, _ := NewManagementClient("session_token", WithBaseURL(server.URL))
			result, err := client.CreateProject(context.Background(), tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateProject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result.Project.ID != tt.response.Project.ID {
					t.Errorf("CreateProject() project ID = %v, want %v",
						result.Project.ID, tt.response.Project.ID)
				}
				if result.APIKey != tt.response.APIKey {
					t.Errorf("CreateProject() API key = %v, want %v",
						result.APIKey, tt.response.APIKey)
				}
			}
		})
	}
}

func TestClient_DeleteProject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		projectID  string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "success",
			projectID:  "proj_test123",
			statusCode: http.StatusNoContent,
			wantErr:    false,
		},
		{
			name:       "not found",
			projectID:  "proj_nonexistent",
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/v1/projects/" + tt.projectID
				if r.URL.Path != expectedPath {
					t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
				}
				if r.Method != "DELETE" {
					t.Errorf("expected DELETE method, got %s", r.Method)
				}

				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client, _ := NewManagementClient("session_token", WithBaseURL(server.URL))
			err := client.DeleteProject(context.Background(), tt.projectID)

			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteProject() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_ListAPIKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		projectID  string
		statusCode int
		response   APIKeyList
		wantErr    bool
	}{
		{
			name:       "success",
			projectID:  "proj_test123",
			statusCode: http.StatusOK,
			response: APIKeyList{
				APIKeys: []APIKey{
					{
						ID:          "key_123",
						ProjectID:   "proj_test123",
						Name:        "Production Key",
						Environment: "live",
						Prefix:      "actlog_live_abc",
						Scopes:      []string{"events:write"},
						CreatedAt:   time.Now(),
					},
				},
			},
			wantErr: false,
		},
		{
			name:       "empty list",
			projectID:  "proj_test123",
			statusCode: http.StatusOK,
			response: APIKeyList{
				APIKeys: []APIKey{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/v1/projects/" + tt.projectID + "/keys"
				if r.URL.Path != expectedPath {
					t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
				}
				if r.Method != "GET" {
					t.Errorf("expected GET method, got %s", r.Method)
				}

				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					json.NewEncoder(w).Encode(tt.response)
				}
			}))
			defer server.Close()

			client, _ := NewManagementClient("session_token", WithBaseURL(server.URL))
			result, err := client.ListAPIKeys(context.Background(), tt.projectID)

			if (err != nil) != tt.wantErr {
				t.Errorf("ListAPIKeys() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(result.APIKeys) != len(tt.response.APIKeys) {
					t.Errorf("ListAPIKeys() returned %d keys, want %d",
						len(result.APIKeys), len(tt.response.APIKeys))
				}
			}
		})
	}
}

func TestClient_CreateAPIKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		projectID  string
		request    CreateAPIKeyRequest
		statusCode int
		response   CreateAPIKeyResponse
		wantErr    bool
	}{
		{
			name:      "success",
			projectID: "proj_test123",
			request: CreateAPIKeyRequest{
				Name:        "New API Key",
				Environment: "test",
				Scopes:      []string{"events:write"},
			},
			statusCode: http.StatusCreated,
			response: CreateAPIKeyResponse{
				APIKeyMetadata: APIKey{
					ID:          "key_new123",
					ProjectID:   "proj_test123",
					Name:        "New API Key",
					Environment: "test",
					Prefix:      "actlog_test_abc",
					Scopes:      []string{"events:write"},
					CreatedAt:   time.Now(),
				},
				APIKey: "actlog_test_1234567890abcdef1234567890abcdef",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/v1/projects/" + tt.projectID + "/keys"
				if r.URL.Path != expectedPath {
					t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
				}
				if r.Method != "POST" {
					t.Errorf("expected POST method, got %s", r.Method)
				}

				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusCreated {
					json.NewEncoder(w).Encode(tt.response)
				}
			}))
			defer server.Close()

			client, _ := NewManagementClient("session_token", WithBaseURL(server.URL))
			result, err := client.CreateAPIKey(context.Background(), tt.projectID, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateAPIKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result.APIKey != tt.response.APIKey {
					t.Errorf("CreateAPIKey() API key = %v, want %v",
						result.APIKey, tt.response.APIKey)
				}
			}
		})
	}
}

func TestClient_RevokeAPIKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		keyID      string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "success",
			keyID:      "key_123",
			statusCode: http.StatusNoContent,
			wantErr:    false,
		},
		{
			name:       "not found",
			keyID:      "key_nonexistent",
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/v1/keys/" + tt.keyID + "/revoke"
				if r.URL.Path != expectedPath {
					t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
				}
				if r.Method != "POST" {
					t.Errorf("expected POST method, got %s", r.Method)
				}

				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client, _ := NewManagementClient("session_token", WithBaseURL(server.URL))
			err := client.RevokeAPIKey(context.Background(), tt.keyID)

			if (err != nil) != tt.wantErr {
				t.Errorf("RevokeAPIKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_RotateAPIKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		keyID      string
		request    RotateAPIKeyRequest
		statusCode int
		response   RotateAPIKeyResponse
		wantErr    bool
	}{
		{
			name:  "success",
			keyID: "key_123",
			request: RotateAPIKeyRequest{
				NewName: "Rotated Key",
			},
			statusCode: http.StatusOK,
			response: RotateAPIKeyResponse{
				NewAPIKeyMetadata: APIKey{
					ID:          "key_new456",
					ProjectID:   "proj_test123",
					Name:        "Rotated Key",
					Environment: "test",
					Prefix:      "actlog_test_xyz",
					Scopes:      []string{"events:write"},
					CreatedAt:   time.Now(),
				},
				NewAPIKey:       "actlog_test_9876543210fedcba9876543210fedcba",
				OldKeyRevokedAt: time.Now(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/v1/keys/" + tt.keyID + "/rotate"
				if r.URL.Path != expectedPath {
					t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
				}
				if r.Method != "POST" {
					t.Errorf("expected POST method, got %s", r.Method)
				}

				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					json.NewEncoder(w).Encode(tt.response)
				}
			}))
			defer server.Close()

			client, _ := NewManagementClient("session_token", WithBaseURL(server.URL))
			result, err := client.RotateAPIKey(context.Background(), tt.keyID, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("RotateAPIKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result.NewAPIKey != tt.response.NewAPIKey {
					t.Errorf("RotateAPIKey() new API key = %v, want %v",
						result.NewAPIKey, tt.response.NewAPIKey)
				}
			}
		})
	}
}
