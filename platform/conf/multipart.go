package conf

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"strings"
)

func NewMultipartUserdata(data string) (*MultipartUserdata, error) {
	m, err := mail.ReadMessage(strings.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("error parsing multipart MIME: %w", err)
	}
	parts, hdrInfo, err := messageToPartsAndHeaderInfo(m)
	if err != nil {
		return nil, fmt.Errorf("error parsing multipart MIME: %w", err)
	}

	buf := &bytes.Buffer{}
	mpWr := multipart.NewWriter(buf)
	mpWr.SetBoundary(hdrInfo.params["boundary"])

	for _, part := range parts {
		partWr, err := mpWr.CreatePart(part.header)
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(partWr, part.body)
		if err != nil {
			return nil, err
		}
	}

	mpMsg := &MultipartUserdata{
		header: hdrInfo,
		parts:  parts,

		writer:       mpWr,
		newMultipart: buf,
	}

	return mpMsg, nil
}

type MultipartUserdata struct {
	// Using the header and parts we can reconstitute the original MIME message
	header headerInfo
	parts  []partInfo

	// We import the parts into a multipart.Writer to allow adding new parts
	writer       *multipart.Writer
	newMultipart *bytes.Buffer
}

func (m *MultipartUserdata) AddPart(header textproto.MIMEHeader, body []byte) error {
	if m.writer == nil {
		return errors.New("cannot add part to read-only multipart")
	}
	partWr, err := m.writer.CreatePart(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(partWr, bytes.NewReader(body))
	if err != nil {
		return err
	}
	return nil
}

func (m *MultipartUserdata) Serialize() (string, error) {
	if m.writer == nil {
		return "", errors.New("cannot serialize read-only multipart")
	}

	err := m.writer.Close()
	if err != nil {
		return "", err
	}
	asStr := &bytes.Buffer{}
	for k, v := range m.header.origHeader {
		asStr.Write([]byte(fmt.Sprintf("%s: %s\n", k, v[0])))
	}
	asStr.Write([]byte("\n"))

	_, err = io.Copy(asStr, m.newMultipart)
	if err != nil {
		return "", err
	}
	return asStr.String(), nil
}

func parseMimeHeader(header mail.Header) (headerInfo, error) {
	contentType := header.Get("Content-Type")
	if contentType == "" {
		return headerInfo{}, errors.New("no Content-Type header found")
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return headerInfo{}, fmt.Errorf("error parsing header: %w", err)
	}
	partTransferEncoding := header.Get("Content-Transfer-Encoding")

	var contentDisposition string
	var fileName string
	contentDispositionHeader := header.Get("Content-Disposition")
	if contentDispositionHeader != "" {
		mediaType, params, err := mime.ParseMediaType(contentDispositionHeader)
		if err != nil {
			return headerInfo{}, fmt.Errorf("error parsing header: %w", err)
		}
		contentDisposition = mediaType
		fileName = params["filename"]
	}
	return headerInfo{
		mediaType:          mediaType,
		params:             params,
		transferEncoding:   partTransferEncoding,
		contentDisposition: contentDisposition,
		fileName:           fileName,
		origHeader:         header,
	}, nil
}

type headerInfo struct {
	mediaType          string
	params             map[string]string
	fileName           string
	contentDisposition string
	transferEncoding   string
	origHeader         mail.Header
}

type partInfo struct {
	header textproto.MIMEHeader
	body   *bytes.Buffer
}

func messageToPartsAndHeaderInfo(m *mail.Message) ([]partInfo, headerInfo, error) {
	hdrInfo, err := parseMimeHeader(m.Header)
	if err != nil {
		return nil, headerInfo{}, fmt.Errorf("error parsing MIME header: %w", err)
	}

	boundary, ok := hdrInfo.params["boundary"]
	if !ok {
		return nil, headerInfo{}, errors.New("no boundary found in MIME header")
	}

	multipartReader := multipart.NewReader(m.Body, boundary)

	parts := []partInfo{}
	for {
		part, err := multipartReader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, headerInfo{}, fmt.Errorf("error reading part: %w", err)
		}
		partHeader := part.Header
		partBody := &bytes.Buffer{}
		_, err = io.Copy(partBody, part)
		if err != nil {
			return nil, headerInfo{}, fmt.Errorf("error reading part: %w", err)
		}
		parts = append(parts, partInfo{
			header: partHeader,
			body:   partBody,
		})
	}
	return parts, hdrInfo, nil
}
