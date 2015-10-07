package gads

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"encoding/xml"
)

const (
	// apiVersion is in base.go
	baseReportAPIURL = "https://adwords.google.com/api/adwords/reportdownload/" + apiVersion

	// DownloadFormatXML is when you want xml in return, eventually parsable in the api in the future
	DownloadFormatXML DownloadFormat = "XML"
	// DownloadFormatXMLGzipped is when you want xml but compressed in gzip format
	DownloadFormatXMLGzipped DownloadFormat = "GZIPPED_XML"
	// DownloadFormatCSV is when you want pure csv in return, with the first line that contains
	DownloadFormatCSV DownloadFormat = "CSV"
	// DownloadFormatCSVGzipped is when you want csv but compressed in gzip
	DownloadFormatCSVGzipped DownloadFormat = "GZIPPED_CSV"
	// DownloadFormatTSV is when you want like csv but separated with tabs
	DownloadFormatTSV DownloadFormat = "TSV"

	// DateRangeTypeCustom is the type used when you specify manually the range of the report
	DateRangeTypeCustom DateRangeType = "CUSTOM_DATE"

	DateRangeTypeToday            DateRangeType = "TODAY"
	DateRangeTypeYesterday        DateRangeType = "YESTERDAY"
	DateRangeTypeLast7Days        DateRangeType = "LAST_7_DAYS"
	DateRangeTypeLastWeek         DateRangeType = "LAST_WEEK"
	DateRangeTypeLastBusinessWeek DateRangeType = "LAST_BUSINESS_WEEK"
	DateRangeTypeThisMonth        DateRangeType = "THIS_MONTH"
	DateRangeTypeAllTime          DateRangeType = "ALL_TIME"
	DateRangeTypeLast14Days       DateRangeType = "LAST_14_DAYS"
	DateRangeTypeLast30Days       DateRangeType = "LAST_30_DAYS"
	DateRangeTypeThisWeekSunToday DateRangeType = "THIS_WEEK_SUN_TODAY"
	DateRangeTypeThisWeekMonToday DateRangeType = "THIS_WEEK_MON_TODAY"
	DateRangeTypeLastWeekSunSat   DateRangeType = "LAST_WEEK_SUN_SAT"
)

// DownloadFormat is the return type of the reports that you want to fetch
type DownloadFormat string

// DateRangeType is the date range when you want
type DateRangeType string

// Valid returns an error if the type is not a part of the allowed DownloadFormat values
func (d DownloadFormat) Valid() error {
	if d != DownloadFormatCSV &&
		d != DownloadFormatXML &&
		d != DownloadFormatCSVGzipped &&
		d != DownloadFormatXMLGzipped &&
		d != DownloadFormatTSV {
		return ErrInvalidReportDownloadType
	}
	return nil
}

// ReportDefinition represents a request for the report API
// https://developers.google.com/adwords/api/docs/guides/reporting
type ReportDefinition struct {
	XMLName                xml.Name       `xml:"reportDefinition"`
	ID                     string         `xml:"id,omitempty"`
	ClientCustomerID       string         `xml:"-"`
	Selector               Selector       `xml:"selector"`
	ReportName             string         `xml:"reportName"`
	ReportType             string         `xml:"reportType"`
	HasAttachment          string         `xml:"hasAttachment,omitempty"`
	DateRangeType          DateRangeType  `xml:"dateRangeType"`
	CreationTime           string         `xml:"creationTime,omitempty"`
	DownloadFormat         DownloadFormat `xml:"downloadFormat"`
	IncludeZeroImpressions bool           `xml:"includeZeroImpressions,omitempty"`
}

// ValidRequest returns an error if the report can't be used to do request to the api
func (r *ReportDefinition) ValidRequest() error {

	if r == nil {
		return errors.New("empty report definition")
	}

	if r.ReportName == "" {
		return ErrMissingReportName
	}
	if r.ReportType == "" {
		return ErrMissingReportType
	}
	if err := r.DownloadFormat.Valid(); err != nil {
		return err
	}

	if r.Selector.DateRange != nil {
		r.DateRangeType = DateRangeTypeCustom
	}

	return nil
}

// ReportDefinitionService is the service you call when you want to access reports
type ReportDefinitionService struct {
	Auth
}

// NewReportDefinitionService creates a ReportDefinitionService that can be accessed with Auth
// object
func NewReportDefinitionService(auth *Auth) *ReportDefinitionService {
	return &ReportDefinitionService{Auth: *auth}
}

// Request launch a request to the reporting api with the definition of the wanted report
// We return a reader because the response format depends of the ReportDefinition.DownloadFormat field
func (r *ReportDefinitionService) Request(def *ReportDefinition) (body io.ReadCloser, err error) {

	var req *http.Request
	req, err = r.createHTTPRequest(def)
	if err != nil {
		return
	}

	var resp *http.Response

	// spec google, some reports can take up to 10 min to be downloaded
	r.Auth.Client.Timeout = 10 * time.Minute

	resp, err = r.Auth.Client.Do(req)
	if err != nil {
		return
	}

	body = resp.Body
	// analyze response code
	if resp.StatusCode != http.StatusOK {
		err = errors.New(fmt.Sprintf("request to report expected Code 200 but received %v ", resp.StatusCode))
		return
	}

	return

}

// createHTTPRequest generates the http request matching the report definition
func (r *ReportDefinitionService) createHTTPRequest(def *ReportDefinition) (req *http.Request, err error) {

	if err = def.ValidRequest(); err != nil {
		return
	}

	var cID string
	if def.ClientCustomerID != "" {
		cID = def.ClientCustomerID
	} else if r.Auth.CustomerId != "" {
		cID = r.Auth.CustomerId
	} else {
		err = ErrMissingCustomerId
		return
	}

	var b []byte
	b, err = xml.Marshal(def)
	if err != nil {
		return nil, err
	}

	var f = url.Values{}
	f.Set("__rdxml", string(b))

	req, err = http.NewRequest("POST", baseReportAPIURL, strings.NewReader(f.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Add("clientCustomerId", cID)
	req.Header.Add("developerToken", r.Auth.DeveloperToken)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	return
}
