package handlers

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ─────────────────────────────────────────────
// Node identity — this node's own keypair
// ─────────────────────────────────────────────

const nodeKeyPath = "/var/lib/fabrixe/node_key.pem"
const nodePubPath = "/var/lib/fabrixe/node_pub.pem"

func ensureNodeKey() (pubKeyPEM, fingerprint string, err error) {
	if data, err := os.ReadFile(nodePubPath); err == nil {
		fp := computePEMFingerprint(string(data))
		return string(data), fp, nil
	}

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("generating key: %w", err)
	}

	privDER, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return "", "", err
	}
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privDER})

	pubDER, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		return "", "", err
	}
	pubPEMBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	_ = os.MkdirAll("/var/lib/fabrixe", 0700)
	_ = os.WriteFile(nodeKeyPath, privPEM, 0600)
	_ = os.WriteFile(nodePubPath, pubPEMBytes, 0644)

	fp := computePEMFingerprint(string(pubPEMBytes))
	return string(pubPEMBytes), fp, nil
}

func computePEMFingerprint(pubPEM string) string {
	block, _ := pem.Decode([]byte(pubPEM))
	if block == nil {
		return "unknown"
	}
	h := sha256.Sum256(block.Bytes)
	var parts []string
	for i, b := range h[:16] {
		if i > 0 {
			parts = append(parts, ":")
		}
		parts = append(parts, fmt.Sprintf("%02x", b))
	}
	return strings.Join(parts, "")
}

// GET /api/comm/identity
func (h *Handler) GetNodeIdentity(w http.ResponseWriter, r *http.Request) {
	pubKey, fp, err := ensureNodeKey()
	if err != nil {
		internalError(w, err)
		return
	}
	hostname, _ := os.Hostname()
	nodeID := getOrCreateNodeID()
	ok(w, map[string]interface{}{
		"node_id":     nodeID,
		"hostname":    hostname,
		"public_key":  pubKey,
		"fingerprint": fp,
		"version":     "1.0.0",
	})
}

func getOrCreateNodeID() string {
	idPath := "/var/lib/fabrixe/.node_id"
	if data, err := os.ReadFile(idPath); err == nil {
		return strings.TrimSpace(string(data))
	}
	id := uuid.NewString()
	_ = os.MkdirAll("/var/lib/fabrixe", 0700)
	_ = os.WriteFile(idPath, []byte(id), 0600)
	return id
}

// ─────────────────────────────────────────────
// Nodes
// ─────────────────────────────────────────────

func (h *Handler) ListNodes(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Conn().Query(`
		SELECT id, node_id, display_name, endpoint, public_key, fingerprint, is_trusted, status, last_seen, created_at, updated_at
		FROM communication_nodes ORDER BY display_name`)
	if err != nil {
		internalError(w, err)
		return
	}
	defer rows.Close()
	var nodes []map[string]interface{}
	for rows.Next() {
		var id int64
		var nodeID, displayName, endpoint, publicKey, fingerprint, status, createdAt, updatedAt string
		var isTrusted int
		var lastSeen sql.NullString
		if err := rows.Scan(&id, &nodeID, &displayName, &endpoint, &publicKey, &fingerprint,
			&isTrusted, &status, &lastSeen, &createdAt, &updatedAt); err != nil {
			continue
		}
		nodes = append(nodes, map[string]interface{}{
			"id": id, "node_id": nodeID, "display_name": displayName,
			"endpoint": endpoint, "public_key": publicKey, "fingerprint": fingerprint,
			"is_trusted": isTrusted == 1, "status": status,
			"last_seen": lastSeen.String, "created_at": createdAt, "updated_at": updatedAt,
		})
	}
	if nodes == nil {
		nodes = []map[string]interface{}{}
	}
	ok(w, nodes)
}

func (h *Handler) GetNode(w http.ResponseWriter, r *http.Request) {
	id, err := paramInt64(r, "id")
	if err != nil {
		badRequest(w, "invalid id")
		return
	}
	var nodeID, displayName, endpoint, publicKey, fingerprint, status, createdAt, updatedAt string
	var isTrusted int
	var lastSeen sql.NullString
	err = h.db.Conn().QueryRow(`
		SELECT node_id, display_name, endpoint, public_key, fingerprint, is_trusted, status, last_seen, created_at, updated_at
		FROM communication_nodes WHERE id = ?`, id).
		Scan(&nodeID, &displayName, &endpoint, &publicKey, &fingerprint, &isTrusted,
			&status, &lastSeen, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		notFound(w)
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	ok(w, map[string]interface{}{
		"id": id, "node_id": nodeID, "display_name": displayName,
		"endpoint": endpoint, "public_key": publicKey, "fingerprint": fingerprint,
		"is_trusted": isTrusted == 1, "status": status,
		"last_seen": lastSeen.String, "created_at": createdAt, "updated_at": updatedAt,
	})
}

type addNodeRequest struct {
	DisplayName string `json:"display_name"`
	Endpoint    string `json:"endpoint"`
	PublicKey   string `json:"public_key"`
}

func (h *Handler) AddNode(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	var req addNodeRequest
	if err := decode(r, &req); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	if req.DisplayName == "" || req.Endpoint == "" || req.PublicKey == "" {
		badRequest(w, "display_name, endpoint, and public_key are required")
		return
	}
	fp := computePEMFingerprint(req.PublicKey)
	rawID := make([]byte, 16)
	_, _ = rand.Read(rawID)
	nodeID := base64.RawURLEncoding.EncodeToString(rawID)
	now := time.Now().UTC().Format(time.RFC3339)

	res, err := h.db.Conn().Exec(`
		INSERT INTO communication_nodes
			(node_id, display_name, endpoint, public_key, fingerprint, is_trusted, status, added_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 0, 'unknown', ?, ?, ?)`,
		nodeID, req.DisplayName, req.Endpoint, req.PublicKey, fp, claims.UserID, now, now)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			fail(w, http.StatusConflict, "node with this fingerprint already exists")
			return
		}
		internalError(w, err)
		return
	}
	newID, _ := res.LastInsertId()
	h.writeAuditLog("comm.node_added",
		fmt.Sprintf("node %q (%s) added by %s", req.DisplayName, fp, claims.Username),
		&claims.UserID, claims.Username, realIP(r), "success")
	created(w, map[string]interface{}{"id": newID, "node_id": nodeID, "fingerprint": fp})
}

func (h *Handler) TrustNode(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	id, _ := paramInt64(r, "id")
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = h.db.Conn().Exec(`UPDATE communication_nodes SET is_trusted=1, updated_at=? WHERE id=?`, now, id)
	h.writeAuditLog("comm.node_trusted",
		fmt.Sprintf("node id=%d trusted by %s", id, claims.Username),
		&claims.UserID, claims.Username, realIP(r), "success")
	ok(w, map[string]string{"message": "node trusted"})
}

func (h *Handler) RevokeNode(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	id, _ := paramInt64(r, "id")
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = h.db.Conn().Exec(`UPDATE communication_nodes SET is_trusted=0, status='unknown', updated_at=? WHERE id=?`, now, id)
	h.writeAuditLog("comm.node_revoked",
		fmt.Sprintf("node id=%d revoked by %s", id, claims.Username),
		&claims.UserID, claims.Username, realIP(r), "success")
	ok(w, map[string]string{"message": "node access revoked"})
}

func (h *Handler) DeleteNode(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	id, _ := paramInt64(r, "id")
	_, _ = h.db.Conn().Exec(`DELETE FROM communication_nodes WHERE id=?`, id)
	h.writeAuditLog("comm.node_deleted",
		fmt.Sprintf("node id=%d deleted by %s", id, claims.Username),
		&claims.UserID, claims.Username, realIP(r), "success")
	ok(w, map[string]string{"message": "node deleted"})
}

// POST /api/comm/ping/{id}
func (h *Handler) PingNode(w http.ResponseWriter, r *http.Request) {
	id, err := paramInt64(r, "id")
	if err != nil {
		badRequest(w, "invalid id")
		return
	}
	var endpoint string
	err = h.db.Conn().QueryRow(`SELECT endpoint FROM communication_nodes WHERE id = ?`, id).Scan(&endpoint)
	if err == sql.ErrNoRows {
		notFound(w)
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}

	start := time.Now()
	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: newInsecureTransport(),
	}
	resp, pingErr := client.Get(strings.TrimRight(endpoint, "/") + "/api/health")
	latencyMs := time.Since(start).Milliseconds()

	status := "online"
	errMsg := ""
	if pingErr != nil {
		status = "offline"
		errMsg = pingErr.Error()
	} else {
		resp.Body.Close()
		if resp.StatusCode >= 500 {
			status = "offline"
			errMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = h.db.Conn().Exec(`UPDATE communication_nodes SET status=?, last_seen=?, updated_at=? WHERE id=?`,
		status, now, now, id)

	result := map[string]interface{}{
		"status":     status,
		"latency_ms": latencyMs,
	}
	if errMsg != "" {
		result["error"] = errMsg
	}
	ok(w, result)
}
