package plugins

import (
	"errors"
	"fmt"

	"github.com/grafana/grafana/pkg/models"
)

const (
	TypeDashboard = "dashboard"
)

var (
	ErrInstallCorePlugin           = errors.New("cannot install a Core plugin")
	ErrUninstallCorePlugin         = errors.New("cannot uninstall a Core plugin")
	ErrUninstallOutsideOfPluginDir = errors.New("cannot uninstall a plugin outside")
	ErrPluginNotInstalled          = errors.New("plugin is not installed")
)

type NotFoundError struct {
	PluginID string
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf("plugin with ID '%s' not found", e.PluginID)
}

type DuplicateError struct {
	PluginID          string
	ExistingPluginDir string
}

func (e DuplicateError) Error() string {
	return fmt.Sprintf("plugin with ID '%s' already exists in '%s'", e.PluginID, e.ExistingPluginDir)
}

func (e DuplicateError) Is(err error) bool {
	// nolint:errorlint
	_, ok := err.(DuplicateError)
	return ok
}

type SignatureError struct {
	PluginID        string          `json:"pluginId"`
	SignatureStatus SignatureStatus `json:"status"`
}

func (e SignatureError) Error() string {
	switch e.SignatureStatus {
	case SignatureInvalid:
		return fmt.Sprintf("plugin '%s' has an invalid signature", e.PluginID)
	case SignatureModified:
		return fmt.Sprintf("plugin '%s' has an modified signature", e.PluginID)
	case SignatureUnsigned:
		return fmt.Sprintf("plugin '%s' has no signature", e.PluginID)
	case SignatureInternal, SignatureValid:
		return ""
	}

	return fmt.Sprintf("plugin '%s' has an unknown signature state", e.PluginID)
}

func (e SignatureError) AsErrorCode() ErrorCode {
	switch e.SignatureStatus {
	case SignatureInvalid:
		return signatureInvalid
	case SignatureModified:
		return signatureModified
	case SignatureUnsigned:
		return signatureMissing
	case SignatureInternal, SignatureValid:
		return ""
	}

	return ""
}

type Dependencies struct {
	GrafanaVersion string       `json:"grafanaVersion"`
	Plugins        []Dependency `json:"plugins"`
}

type Includes struct {
	Name       string          `json:"name"`
	Path       string          `json:"path"`
	Type       string          `json:"type"`
	Component  string          `json:"component"`
	Role       models.RoleType `json:"role"`
	AddToNav   bool            `json:"addToNav"`
	DefaultNav bool            `json:"defaultNav"`
	Slug       string          `json:"slug"`
	Icon       string          `json:"icon"`
	UID        string          `json:"uid"`

	ID string `json:"-"`
}

func (e Includes) GetSlugOrUIDLink() string {
	if len(e.UID) > 0 {
		return "/d/" + e.UID
	} else {
		return "/dashboard/db/" + e.Slug
	}
}

type Dependency struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type BuildInfo struct {
	Time   int64  `json:"time,omitempty"`
	Repo   string `json:"repo,omitempty"`
	Branch string `json:"branch,omitempty"`
	Hash   string `json:"hash,omitempty"`
}

type Info struct {
	Author      InfoLink      `json:"author"`
	Description string        `json:"description"`
	Links       []InfoLink    `json:"links"`
	Logos       Logos         `json:"logos"`
	Build       BuildInfo     `json:"build"`
	Screenshots []Screenshots `json:"screenshots"`
	Version     string        `json:"version"`
	Updated     string        `json:"updated"`
}

type InfoLink struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Logos struct {
	Small string `json:"small"`
	Large string `json:"large"`
}

type Screenshots struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type StaticRoute struct {
	PluginID  string
	Directory string
}

type SignatureStatus string

func (ss SignatureStatus) IsValid() bool {
	return ss == SignatureValid
}

func (ss SignatureStatus) IsInternal() bool {
	return ss == SignatureInternal
}

const (
	SignatureInternal SignatureStatus = "internal" // core plugin, no signature
	SignatureValid    SignatureStatus = "valid"    // signed and accurate MANIFEST
	SignatureInvalid  SignatureStatus = "invalid"  // invalid signature
	SignatureModified SignatureStatus = "modified" // valid signature, but content mismatch
	SignatureUnsigned SignatureStatus = "unsigned" // no MANIFEST file
)

type ReleaseState string

const (
	AlphaRelease ReleaseState = "alpha"
)

type SignatureType string

const (
	GrafanaSignature SignatureType = "grafana"
	PrivateSignature SignatureType = "private"
)

type PluginFiles map[string]struct{}

type Signature struct {
	Status     SignatureStatus
	Type       SignatureType
	SigningOrg string
	Files      PluginFiles
}

type PluginMetaDTO struct {
	JSONData

	Signature SignatureStatus `json:"signature"`

	Module  string `json:"module"`
	BaseURL string `json:"baseUrl"`
}

type DataSourceDTO struct {
	ID        int64                  `json:"id"`
	UID       string                 `json:"uid"`
	Type      string                 `json:"Type"`
	Name      string                 `json:"name"`
	URL       string                 `json:"url"`
	IsDefault bool                   `json:"isDefault"`
	Access    string                 `json:"access"`
	Preload   bool                   `json:"preload"`
	Module    string                 `json:"module"`
	Meta      *PluginMetaDTO         `json:"meta"`
	JSONData  map[string]interface{} `json:"jsonData"`

	BasicAuth       string `json:"basicAuth"`
	WithCredentials bool   `json:"withCredentials"`

	// InfluxDB
	Username string `json:"username"`
	Password string `json:"password"`

	// InfluxDB + Elasticsearch
	Database string `json:"database"`

	// Prometheus
	DirectURL string `json:"directUrl"`
}

const (
	signatureMissing  ErrorCode = "signatureMissing"
	signatureModified ErrorCode = "signatureModified"
	signatureInvalid  ErrorCode = "signatureInvalid"
)

type ErrorCode string

type Error struct {
	ErrorCode `json:"errorCode"`
	PluginID  string `json:"pluginId,omitempty"`
}
