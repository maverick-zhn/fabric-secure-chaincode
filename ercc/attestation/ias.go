/*
* Copyright IBM Corp. 2018 All Rights Reserved.
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*     http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package attestation

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

// intel verification key
const IntelPubPEM = `
-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAqXot4OZuphR8nudFrAFi
aGxxkgma/Es/BA+tbeCTUR106AL1ENcWA4FX3K+E9BBL0/7X5rj5nIgX/R/1ubhk
KWw9gfqPG3KeAtIdcv/uTO1yXv50vqaPvE1CRChvzdS/ZEBqQ5oVvLTPZ3VEicQj
lytKgN9cLnxbwtuvLUK7eyRPfJW/ksddOzP8VBBniolYnRCD2jrMRZ8nBM2ZWYwn
XnwYeOAHV+W9tOhAImwRwKF/95yAsVwd21ryHMJBcGH70qLagZ7Ttyt++qO/6+KA
XJuKwZqjRlEtSEz8gZQeFfVYgcwSfo96oSMAzVr7V0L6HSDLRnpb6xxmbPdqNol4
tQIDAQAB
-----END PUBLIC KEY-----`

const iasURL = "https://test-as.sgx.trustedservices.intel.com:443/attestation/sgx/v2/report"

// IASReportBody received from IAS (Intel attestation service)
type IASReportBody struct {
	ID                    string `json:"id"`
	IsvEnclaveQuoteStatus string `json:"isvEnclaveQuoteStatus"`
	IsvEnclaveQuoteBody   string `json:"isvEnclaveQuoteBody"`
	PlatformInfoBlob      string `json:"platformInfoBlob,omitempty"`
	RevocationReason      string `json:"revocationReason,omitempty"`
	PseManifestStatus     string `json:"pseManifestStatus,omitempty"`
	PseManifestHash       string `json:"pseManifestHash,omitempty"`
	Nonce                 string `json:"nonce,omitempty"`
	EpidPseudonym         string `json:"epidPseudonym,omitempty"`
	Timestamp             string `json:"timestamp"`
}

// IASAttestationReport received from IAS (Intel attestation service)
// TODO renamte to AttestationReport
type IASAttestationReport struct {
	EnclavePk                   []byte `json:"EnclavePk"`
	IASReportSignature          string `json:"IASReport-Signature"`
	IASReportSigningCertificate string `json:"IASReport-Signing-Certificate"`
	IASReportBody               []byte `json:"IASResponseBody"`
}

// IntelAttestationService sent to IAS (Intel attestation service)
type IntelAttestationService interface {
	RequestAttestationReport(cert tls.Certificate, quoteAsBytes []byte) (IASAttestationReport, error)
	GetIntelVerificationKey() (interface{}, error)
}

type intelAttestationServiceImpl struct {
	url string
}

// NewIAS is a great help to build an IntelAttestationService object
func NewIAS() IntelAttestationService {
	return &intelAttestationServiceImpl{url: iasURL}
}

// RequestAttestationReport sends a quote to Intel for verification and in return receives an IASAttestationReport
// Calling Intel qualifies ercc as a system chaincode since in the future chaincodes might be restricted and can not make call outside their docker container
func (ias *intelAttestationServiceImpl) RequestAttestationReport(cert tls.Certificate, quoteAsBytes []byte) (IASAttestationReport, error) {

	// Setup HTTPS client
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		// RootCAs:            caCertPool,
		InsecureSkipVerify: true,
	}
	tlsConfig.BuildNameToCertificate()
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: transport}

	// transform quote bytes to base64 and build request body
	quoteAsBase64 := base64.StdEncoding.EncodeToString(quoteAsBytes)
	requestBody := &IASRequestBody{Quote: quoteAsBase64}
	requestBytes, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", ias.url, bytes.NewBuffer(requestBytes))
	if err != nil {
		return IASAttestationReport{}, fmt.Errorf("IAS connection error: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")

	// submit quote for verification
	resp, err := client.Do(req)
	if err != nil {
		return IASAttestationReport{}, fmt.Errorf("IAS connection error: %s", err)
	}
	defer resp.Body.Close()

	// check response
	if resp.StatusCode != 200 {
		return IASAttestationReport{}, fmt.Errorf("IAS returned error: Code %s", resp.Status)
	}

	bodyData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return IASAttestationReport{}, fmt.Errorf("Can not read response body: %s", err)
	}

	reportBody := IASReportBody{}
	json.Unmarshal(bodyData, &reportBody)

	// check response contains submitted quote
	if !strings.HasPrefix(quoteAsBase64, reportBody.IsvEnclaveQuoteBody) {
		return IASAttestationReport{}, errors.New("Report does not contain submitted quote")
	}

	report := IASAttestationReport{
		IASReportSignature:          resp.Header.Get("X-IASReport-Signature"),
		IASReportSigningCertificate: resp.Header.Get("X-IASReport-Signing-Certificate"),
		IASReportBody:               bodyData,
	}

	return report, nil
}

func (ias *intelAttestationServiceImpl) GetIntelVerificationKey() (interface{}, error) {
	return PublicKeyFromPem([]byte(IntelPubPEM))
}

func PublicKeyFromPem(bytes []byte) (interface{}, error) {
	block, _ := pem.Decode([]byte(bytes))
	if block == nil {
		return nil, fmt.Errorf("Failed to parse PEM block containing the public key")
	}
	pk, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("Public key is invalid: %s", err)
	}
	return pk, nil
}
