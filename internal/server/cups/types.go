package cups

import (
	"net/http"
	"time"
	"sync"
	"context"
	
	"github.com/godbus/dbus/v5"
)

type CUPSState struct {
	Printers	 map[string]*Printer `json:"printers"`
}

type Printer struct {
	Name        string  `json:"name"`
	URI         string  `json:"uri"`
	State       string  `json:"state"`
	StateReason string  `json:"stateReason"`
	Location    string  `json:"location"`
	Info        string  `json:"info"`
	MakeModel   string  `json:"makeModel"`
	Accepting   bool    `json:"accepting"`
	Jobs        []Job   `json:"jobs"`
}

type Job struct {
	ID          int         `json:"id"`
	Name        string      `json:"name"`
	State       string      `json:"state"`
	Printer     string      `json:"printer"`
	User        string      `json:"user"`
	Size        int         `json:"size"`
	TimeCreated time.Time   `json:"timeCreated"`
}

type Manager struct {
	state          *CUPSState
	BaseURL        string
	Client         *http.Client
	stateMutex     sync.RWMutex
	subscribers    map[string]chan CUPSState
	subMutex       sync.RWMutex
	stopChan       chan struct{}
	dbusConn       *dbus.Conn
	signals        chan *dbus.Signal
	sigWG          sync.WaitGroup
	dirty          chan struct{}
	notifierWg     sync.WaitGroup
	lastNotifiedState  *CUPSState
	lm             LogMonitor
}

// log fallback
type LogMonitor struct {
	ctx           context.Context
	cancel        context.CancelFunc
	logPaths      []string
	manager       *Manager
}

type CUPSAccessLogEntry struct {
	Host         string
	Group        string
	User         string
	Timestamp    time.Time
	Method       string
	Resource     string
	Version      string
	Status       int
	Bytes        int
	IPPOperation string
	IPPStatus    string
}

// IPP Operation IDs
const (
	IPP_OP_PRINT_JOB           = 0x0002
	IPP_OP_VALIDATE_JOB        = 0x0004
	IPP_OP_GET_PRINTER_ATTRS   = 0x000B
	IPP_OP_GET_JOBS            = 0x000A
	IPP_OP_CANCEL_JOB          = 0x0008
	IPP_OP_PAUSE_PRINTER       = 0x0010
	IPP_OP_RESUME_PRINTER      = 0x0011
	IPP_OP_PURGE_JOBS          = 0x0012
	IPP_OP_CUPS_GET_PRINTERS   = 0x4002
	IPP_OP_CUPS_GET_DEFAULT    = 0x4001
)

// IPP Status Codes
const (
	IPP_STATUS_OK                = 0x0000
	IPP_STATUS_CLIENT_ERROR      = 0x0400
	IPP_STATUS_SERVER_ERROR      = 0x0500
)

// IPP Tags
const (
	IPP_TAG_ZERO              = 0x00
	IPP_TAG_OPERATION         = 0x01
	IPP_TAG_JOB               = 0x02
	IPP_TAG_END               = 0x03
	IPP_TAG_PRINTER           = 0x04
	IPP_TAG_UNSUPPORTED_GROUP = 0x05
	IPP_TAG_SUBSCRIPTION      = 0x06
	IPP_TAG_EVENT_NOTIFICATION = 0x07
	IPP_TAG_INTEGER           = 0x21
	IPP_TAG_BOOLEAN           = 0x22
	IPP_TAG_ENUM              = 0x23
	IPP_TAG_STRING            = 0x30
	IPP_TAG_DATE              = 0x31
	IPP_TAG_RESOLUTION        = 0x32
	IPP_TAG_RANGE             = 0x33
	IPP_TAG_BEGIN_COLLECTION = 0x34
	IPP_TAG_TEXT_LANG         = 0x35
	IPP_TAG_NAME_LANG         = 0x36
	IPP_TAG_END_COLLECTION    = 0x37
	IPP_TAG_TEXT              = 0x41
	IPP_TAG_NAME              = 0x42
	IPP_TAG_KEYWORD           = 0x44
	IPP_TAG_URI               = 0x45
	IPP_TAG_CHARSET           = 0x47
	IPP_TAG_LANGUAGE          = 0x48
	IPP_TAG_MIMETYPE          = 0x49
)
