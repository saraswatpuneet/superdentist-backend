package gsheets

import (
	"fmt"
	"os"
	"time"

	"github.com/superdentist/superdentist-backend/contracts"
	"github.com/superdentist/superdentist-backend/lib/helpers"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// Client ....
type Client struct {
	projectID string
	client    *sheets.Service
}

// NewSheetsHandler return new database action
func NewSheetsHandler() *Client {
	return &Client{projectID: "", client: nil}
}

// InitializeSheetsClient ...........
func (sc *Client) InitializeSheetsClient(ctx context.Context, projectID string) error {
	serviceAccountSD := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if serviceAccountSD == "" {
		return fmt.Errorf("Failed to get right credentials for superdentist backend")
	}
	targetScopes := []string{
		"https://www.googleapis.com/auth/cloud-platform",
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/spreadsheets",
		"https://www.googleapis.com/auth/drive",
	}
	currentCreds, _, err := helpers.ReadCredentialsFile(ctx, serviceAccountSD, targetScopes)
	if err != nil {
		return err
	}
	client, err := sheets.NewService(ctx, option.WithCredentials(currentCreds))
	if err != nil {
		return err
	}
	sc.client = client
	sc.projectID = projectID
	return nil
}

func (sc *Client) WritePatientoGSheet(patient contracts.Patient, sheetID string) error {
	var vr sheets.ValueRange
	var pValues []interface{}
	pValues = append(pValues, patient.CreationDate)
	pValues = append(pValues, patient.CreationDate)
	if patient.ClinicName != "" {
		pValues = append(pValues, patient.ClinicName)

	} else if patient.GDName != "" {
		pValues = append(pValues, patient.GDName+"Referred To "+patient.SPName)
	}
	pValues = append(pValues, patient.FirstName)
	pValues = append(pValues, patient.LastName)
	pValues = append(pValues, patient.Dob.Month+"/"+patient.Dob.Day+"/"+patient.Dob.Year)
	pValues = append(pValues, patient.ZipCode)
	if len(patient.DentalInsurance) > 0 {
		currentDI := "Dental Insurances: "
		for _, dI := range patient.DentalInsurance {
			currentDI += " Company: " + dI.Company + " MemberID: " + dI.MemberID
			if dI.Subscriber.FirstName != "" {
				currentDI += " Subscriber Name: " + dI.Subscriber.FirstName + " " + dI.Subscriber.LastName
				currentDI += " Subscriber DOB: " + dI.Subscriber.DOB.Month + "/" + dI.Subscriber.DOB.Day + "/" + dI.Subscriber.DOB.Year

			}
		}
		pValues = append(pValues, currentDI)
	} else {
		pValues = append(pValues, "Dental Insurances: Missing")
	}
	if len(patient.MedicalInsurance) > 0 {
		currentMI := " Medical Insurances: "
		for _, dI := range patient.MedicalInsurance {
			currentMI += "Company: " + dI.Company + " MemberID: " + dI.MemberID + " SSN: " + dI.SSN
			if dI.Subscriber.FirstName != "" {
				currentMI += " Subscriber Name: " + dI.Subscriber.FirstName + " " + dI.Subscriber.LastName
				currentMI += " Subscriber DOB: " + dI.Subscriber.DOB.Month + "/" + dI.Subscriber.DOB.Day + "/" + dI.Subscriber.DOB.Year

			}
		}
		pValues = append(pValues, currentMI)
	} else {
		pValues = append(pValues, "Medical Insurances: Missing")
	}
	if patient.DueDate> 0 {
		pValues = append(pValues, time.Unix(patient.DueDate*int64(time.Millisecond), 0).String())
	} else {
		pValues = append(pValues, time.Now().String())

	}
	vr.Values = append(vr.Values, pValues)
	vr.MajorDimension = "ROWS"
	rangeWrite := "Patients!A1:A1"
	_, err := sc.client.Spreadsheets.Values.Append(sheetID, rangeWrite, &vr).ValueInputOption("RAW").Do()
	if err != nil {
		return err
	}
	return nil
}
