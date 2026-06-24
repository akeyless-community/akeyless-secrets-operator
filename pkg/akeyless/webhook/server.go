/*
Copyright Akeyless Community

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package webhook receives Akeyless change notifications and triggers sync.
package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	akeylessv1alpha1 "github.com/external-secrets/external-secrets/apis/akeyless/v1alpha1"
)

const defaultPath = "/webhook/akeyless"

// Event is the JSON payload from an Akeyless notification webhook.
type Event struct {
	ItemName    string `json:"item_name"`
	Path        string `json:"path"`
	ItemPath    string `json:"item_path"`
	EventType   string `json:"event_type"`
	LastVersion int32  `json:"last_version,omitempty"`
}

// Server handles inbound Akeyless events.
type Server struct {
	Client client.Client
	Log    logr.Logger
	Addr   string
	Token  string
	Path   string
}

// Start implements manager.Runnable.
func (s *Server) Start(ctx context.Context) error {
	path := s.Path
	if path == "" {
		path = defaultPath
	}

	mux := http.NewServeMux()
	mux.HandleFunc(path, s.handleEvent)

	srv := &http.Server{
		Addr:              s.Addr,
	Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	s.Log.Info("starting Akeyless event webhook", "addr", s.Addr, "path", path)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) handleEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.Token != "" {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+s.Token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	var event Event
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	itemPath := event.ItemName
	if itemPath == "" {
		itemPath = event.Path
	}
	if itemPath == "" {
		itemPath = event.ItemPath
	}
	if itemPath == "" {
		http.Error(w, "missing item path", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	count, err := s.enqueueMatching(ctx, itemPath)
	if err != nil {
		s.Log.Error(err, "failed to enqueue sync", "item", itemPath)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	s.Log.Info("Akeyless event processed", "item", itemPath, "event_type", event.EventType, "enqueued", count)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"enqueued": count,
		"item":     itemPath,
	})
}

func (s *Server) enqueueMatching(ctx context.Context, itemPath string) (int, error) {
	count := 0

	var secrets akeylessv1alpha1.AkeylessSecretList
	if err := s.Client.List(ctx, &secrets); err != nil {
		return 0, fmt.Errorf("list AkeylessSecret: %w", err)
	}
	for i := range secrets.Items {
		if !referencesPath(secrets.Items[i].Spec.Data, itemPath) {
			continue
		}
		if err := s.patchForceSync(ctx, &secrets.Items[i]); err != nil {
			return count, err
		}
		count++
	}

	var dynamic akeylessv1alpha1.AkeylessDynamicSecretList
	if err := s.Client.List(ctx, &dynamic); err != nil {
		return count, fmt.Errorf("list AkeylessDynamicSecret: %w", err)
	}
	for i := range dynamic.Items {
		if !pathsEqual(dynamic.Items[i].Spec.Path, itemPath) {
			continue
		}
		if err := s.patchForceSync(ctx, &dynamic.Items[i]); err != nil {
			return count, err
		}
		count++
	}

	return count, nil
}

func (s *Server) patchForceSync(ctx context.Context, obj client.Object) error {
	now := time.Now().Format(time.RFC3339)
	original := obj.DeepCopyObject().(client.Object)
	patch := client.MergeFrom(original)
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[akeylessv1alpha1.AnnotationForceSync] = now
	obj.SetAnnotations(annotations)
	return s.Client.Patch(ctx, obj, patch)
}

func referencesPath(data []akeylessv1alpha1.SecretMapping, itemPath string) bool {
	for _, mapping := range data {
		if pathsEqual(mapping.RemoteRef.Key, itemPath) {
			return true
		}
	}
	return false
}

func pathsEqual(a, b string) bool {
	return normalizePath(a) == normalizePath(b)
}

func normalizePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return p
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return strings.TrimSuffix(p, "/")
}
