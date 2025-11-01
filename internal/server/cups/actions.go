package cups

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func (m *Manager) GetPrinters() ([]Printer, error) {
	attrs := map[string]interface{}{
		"requested-attributes": []string{
			"printer-name",
			"printer-uri-supported",
			"printer-state",
			"printer-state-reasons",
			"printer-location",
			"printer-info",
			"printer-make-and-model",
			"printer-is-accepting-jobs",
		},
	}

	reqBuf := m.buildIPPRequest(IPP_OP_CUPS_GET_PRINTERS, m.BaseURL+"/", attrs)
	groups, err := m.sendIPPRequest(reqBuf)
	if err != nil {
		return nil, err
	}

	printers := []Printer{}

	for i, group := range groups {
		if i == 0 {
			continue
		}

		printer := Printer{}

		for name, value := range group {
			val := value
			if arr, ok := value.([]interface{}); ok && len(arr) > 0 {
				val = arr[0]
			}

			switch name {
			case "printer-name":
				printer.Name = fmt.Sprintf("%v", val)
			case "printer-uri-supported":
				printer.URI = fmt.Sprintf("%v", val)
			case "printer-state":
				state := fmt.Sprintf("%v", val)
				switch state {
				case "3":
					printer.State = "idle"
				case "4":
					printer.State = "processing"
				case "5":
					printer.State = "stopped"
				default:
					printer.State = state
				}
			case "printer-state-reasons":
				printer.StateReason = fmt.Sprintf("%v", val)
			case "printer-location":
				printer.Location = fmt.Sprintf("%v", val)
			case "printer-info":
				printer.Info = fmt.Sprintf("%v", val)
			case "printer-make-and-model":
				printer.MakeModel = fmt.Sprintf("%v", val)
			case "printer-is-accepting-jobs":
				printer.Accepting = val == true || fmt.Sprintf("%v", val) == "true" || fmt.Sprintf("%v", val) == "1"
			}
		}

		if printer.Name != "" {
			printers = append(printers, printer)
		}
	}

	return printers, nil
}

func (m *Manager) GetJobs(printerName string, whichJobs string) ([]Job, error) {
	printerURI := fmt.Sprintf("%s/printers/%s", m.BaseURL, printerName)

	attrs := map[string]interface{}{
		"which-jobs": whichJobs, // "completed", "not-completed", o "all"
		"my-jobs":    false,
		"requested-attributes": []string{
			"job-id",
			"job-name",
			"job-state",
			"job-printer-uri",
			"job-originating-user-name",
			"job-k-octets",
			"time-at-creation",
		},
	}

	reqBuf := m.buildIPPRequest(IPP_OP_GET_JOBS, printerURI, attrs)
	groups, err := m.sendIPPRequest(reqBuf)
	if err != nil {
		return nil, err
	}

	return m.parseJobGroups(groups), nil
}

func (m *Manager) parseJobGroups(groups []map[string]interface{}) []Job {
	jobs := []Job{}

	for i, group := range groups {
		if i == 0 {
			continue
		}

		job := Job{}

		for name, value := range group {
			val := value
			if arr, ok := value.([]interface{}); ok && len(arr) > 0 {
				val = arr[0]
			}

			switch name {
			case "job-id":
				if v, ok := val.(int); ok {
					job.ID = v
				}
			case "job-name":
				job.Name = fmt.Sprintf("%v", val)
			case "job-state":
				state := fmt.Sprintf("%v", val)
				switch state {
				case "3":
					job.State = "pending"
				case "4":
					job.State = "pending-held"
				case "5":
					job.State = "processing"
				case "6":
					job.State = "processing-stopped"
				case "7":
					job.State = "canceled"
				case "8":
					job.State = "aborted"
				case "9":
					job.State = "completed"
				default:
					job.State = state
				}
			case "job-printer-uri":
				uri := fmt.Sprintf("%v", val)
				parts := strings.Split(uri, "/")
				if len(parts) > 0 {
					job.Printer = parts[len(parts)-1]
				}
			case "job-originating-user-name":
				job.User = fmt.Sprintf("%v", val)
			case "job-k-octets":
				if v, ok := val.(int); ok {
					job.Size = v * 1024
				}
			case "time-at-creation":
				if v, ok := val.(int); ok {
					job.TimeCreated = time.Unix(int64(v), 0)
				}
			}
		}

		if job.ID != 0 {
			jobs = append(jobs, job)
		}
	}

	return jobs
}

func (m *Manager) CancelJob(printerName string, jobID int) error {
	jobURI := fmt.Sprintf("%s/jobs/%d", m.BaseURL, jobID)

	attrs := map[string]interface{}{}
	reqBuf := m.buildIPPRequest(IPP_OP_CANCEL_JOB, jobURI, attrs)

	_, err := m.sendIPPRequest(reqBuf)
	return err
}

func (m *Manager) PausePrinter(printerName string) error {
	printerURI := fmt.Sprintf("%s/printers/%s", m.BaseURL, printerName)

	attrs := map[string]interface{}{}
	reqBuf := m.buildIPPRequest(IPP_OP_PAUSE_PRINTER, printerURI, attrs)

	_, err := m.sendIPPRequest(reqBuf)
	return err
}

func (m *Manager) ResumePrinter(printerName string) error {
	printerURI := fmt.Sprintf("%s/printers/%s", m.BaseURL, printerName)

	attrs := map[string]interface{}{}
	reqBuf := m.buildIPPRequest(IPP_OP_RESUME_PRINTER, printerURI, attrs)

	_, err := m.sendIPPRequest(reqBuf)
	return err
}

func (m *Manager) PurgeJobs(printerName string) error {
	printerURI := fmt.Sprintf("%s/printers/%s", m.BaseURL, printerName)

	attrs := map[string]interface{}{}
	reqBuf := m.buildIPPRequest(IPP_OP_PURGE_JOBS, printerURI, attrs)

	_, err := m.sendIPPRequest(reqBuf)
	return err
}

func (m *Manager) buildIPPRequest(operation uint16, uri string, attrs map[string]interface{}) *bytes.Buffer {
	buf := new(bytes.Buffer)

	// IPP Version (2.0)
	binary.Write(buf, binary.BigEndian, uint8(2))
	binary.Write(buf, binary.BigEndian, uint8(0))

	// Operation ID
	binary.Write(buf, binary.BigEndian, operation)

	// Request ID
	binary.Write(buf, binary.BigEndian, uint32(1))

	// Operation attributes group
	buf.WriteByte(IPP_TAG_OPERATION)

	// attributes-charset
	m.writeAttribute(buf, IPP_TAG_CHARSET, "attributes-charset", []byte("utf-8"))

	// attributes-natural-language
	m.writeAttribute(buf, IPP_TAG_LANGUAGE, "attributes-natural-language", []byte("en"))

	// printer-uri o job-uri
	m.writeAttribute(buf, IPP_TAG_URI, "printer-uri", []byte(uri))

	// requesting-user-name
	m.writeAttribute(buf, IPP_TAG_NAME, "requesting-user-name", []byte("cups-go-client"))

	// Attributi aggiuntivi
	for name, value := range attrs {
		switch v := value.(type) {
		case string:
			m.writeAttribute(buf, IPP_TAG_NAME, name, []byte(v))
		case int:
			m.writeIntegerAttribute(buf, name, int32(v))
		case bool:
			m.writeBooleanAttribute(buf, name, v)
		case []string:
			for i, s := range v {
				if i == 0 {
					m.writeAttribute(buf, IPP_TAG_KEYWORD, name, []byte(s))
				} else {
					m.writeAttribute(buf, IPP_TAG_KEYWORD, "", []byte(s))
				}
			}
		}
	}

	// End of attributes
	buf.WriteByte(IPP_TAG_END)

	return buf
}

func (m *Manager) writeAttribute(buf *bytes.Buffer, tag byte, name string, value []byte) {
	buf.WriteByte(tag)
	binary.Write(buf, binary.BigEndian, uint16(len(name)))
	buf.WriteString(name)
	binary.Write(buf, binary.BigEndian, uint16(len(value)))
	buf.Write(value)
}

func (m *Manager) writeIntegerAttribute(buf *bytes.Buffer, name string, value int32) {
	buf.WriteByte(IPP_TAG_INTEGER)
	binary.Write(buf, binary.BigEndian, uint16(len(name)))
	buf.WriteString(name)
	binary.Write(buf, binary.BigEndian, uint16(4))
	binary.Write(buf, binary.BigEndian, value)
}

func (m *Manager) writeBooleanAttribute(buf *bytes.Buffer, name string, value bool) {
	buf.WriteByte(IPP_TAG_BOOLEAN)
	binary.Write(buf, binary.BigEndian, uint16(len(name)))
	buf.WriteString(name)
	binary.Write(buf, binary.BigEndian, uint16(1))
	if value {
		buf.WriteByte(1)
	} else {
		buf.WriteByte(0)
	}
}

func (m *Manager) sendIPPRequest(reqBuf *bytes.Buffer) ([]map[string]interface{}, error) {
	req, err := http.NewRequest("POST", m.BaseURL, reqBuf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/ipp")

	resp, err := m.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	return m.parseIPPResponse(resp.Body)
}

func (m *Manager) parseIPPResponse(r io.Reader) ([]map[string]interface{}, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if len(data) < 8 {
		return nil, fmt.Errorf("risposta IPP troppo corta")
	}

	statusCode := binary.BigEndian.Uint16(data[2:4])

	if statusCode != IPP_STATUS_OK && statusCode < IPP_STATUS_CLIENT_ERROR {
		// valid anyway
	} else if statusCode >= IPP_STATUS_CLIENT_ERROR {
		return nil, fmt.Errorf("errore IPP: 0x%04x", statusCode)
	}

	groups := []map[string]interface{}{}
	currentGroup := make(map[string]interface{})
	pos := 8
	lastAttrName := ""

	for pos < len(data) {
		tag := data[pos]
		pos++

		if tag == IPP_TAG_END {
			if len(currentGroup) > 0 {
				groups = append(groups, currentGroup)
			}
			break
		}

		if tag == IPP_TAG_OPERATION || tag == IPP_TAG_JOB || tag == IPP_TAG_PRINTER {
			if len(currentGroup) > 0 {
				groups = append(groups, currentGroup)
				currentGroup = make(map[string]interface{})
			}
			continue
		}

		if pos+2 > len(data) {
			break
		}

		nameLen := int(binary.BigEndian.Uint16(data[pos : pos+2]))
		pos += 2

		if pos+nameLen > len(data) {
			break
		}

		name := string(data[pos : pos+nameLen])
		pos += nameLen

		if pos+2 > len(data) {
			break
		}

		valueLen := int(binary.BigEndian.Uint16(data[pos : pos+2]))
		pos += 2

		if pos+valueLen > len(data) {
			break
		}

		value := data[pos : pos+valueLen]
		pos += valueLen

		if name == "" {
			name = lastAttrName
		} else {
			lastAttrName = name
		}

		if name != "" {
			parsedValue := m.parseIPPValue(tag, value)

			if existing, ok := currentGroup[name]; ok {
				switch existingVal := existing.(type) {
				case []interface{}:
					currentGroup[name] = append(existingVal, parsedValue)
				default:
					currentGroup[name] = []interface{}{existingVal, parsedValue}
				}
			} else {
				currentGroup[name] = parsedValue
			}
		}
	}

	return groups, nil
}

func (m *Manager) parseIPPValue(tag byte, value []byte) interface{} {
	switch tag {
	case IPP_TAG_INTEGER, IPP_TAG_ENUM:
		if len(value) == 4 {
			return int(binary.BigEndian.Uint32(value))
		}
	case IPP_TAG_BOOLEAN:
		if len(value) == 1 {
			return value[0] != 0
		}
	case IPP_TAG_TEXT, IPP_TAG_NAME, IPP_TAG_KEYWORD, IPP_TAG_URI, IPP_TAG_CHARSET, IPP_TAG_LANGUAGE, IPP_TAG_MIMETYPE:
		return string(value)
	}
	return string(value)
}
