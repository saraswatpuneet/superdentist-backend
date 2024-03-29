package datastoredb

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/superdentist/superdentist-backend/contracts"
	"github.com/superdentist/superdentist-backend/global"
	"github.com/superdentist/superdentist-backend/lib/helpers"
	"google.golang.org/api/option"

	"cloud.google.com/go/datastore"
)

// DSReferral ...
type DSReferral struct {
	projectID string
	client    *datastore.Client
}

//NewReferralHandler return new database action
func NewReferralHandler() *DSReferral {
	return &DSReferral{projectID: "", client: nil}
}

// InitializeDataBase ....
func (db *DSReferral) InitializeDataBase(ctx context.Context, projectID string) error {
	serviceAccountSD := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if serviceAccountSD == "" {
		return fmt.Errorf("Failed to get right credentials for superdentist backend")
	}
	targetScopes := []string{
		"https://www.googleapis.com/auth/cloud-platform",
		"https://www.googleapis.com/auth/userinfo.email",
	}
	currentCreds, _, err := helpers.ReadCredentialsFile(ctx, serviceAccountSD, targetScopes)
	dsClient, err := datastore.NewClient(context.Background(), projectID, option.WithCredentials(currentCreds))
	if err != nil {
		return err
	}
	// Verify that we can communicate and authenticate with the datastore service.
	t, err := dsClient.NewTransaction(ctx)
	if err != nil {
		return fmt.Errorf("datastoredb: could not connect: %v", err)
	}
	if err := t.Rollback(); err != nil {
		return fmt.Errorf("datastoredb: could not connect: %v", err)
	}
	db.client = dsClient
	db.projectID = projectID
	return nil
}

// CreateReferral .....
func (db *DSReferral) CreateReferral(ctx context.Context, referral contracts.DSReferral) error {
	primaryKey := datastore.NameKey("ClinicReferrals", referral.ReferralID, nil)
	if global.Options.DSName != "" {
		primaryKey.Namespace = global.Options.DSName
	}
	//lets create the clinic
	_, err := db.client.Put(ctx, primaryKey, &referral)
	if err != nil {
		return fmt.Errorf("cannot register clinic with sd: %v", err)
	}
	return nil
}

// GetReferral .....
func (db *DSReferral) GetReferral(ctx context.Context, refID string) (*contracts.DSReferral, error) {
	primaryKey := datastore.NameKey("ClinicReferrals", refID, nil)
	if global.Options.DSName != "" {
		primaryKey.Namespace = global.Options.DSName
	}
	var referral contracts.DSReferral
	if global.Options.DSName != "" {
		primaryKey.Namespace = global.Options.DSName
	}
	err := db.client.Get(ctx, primaryKey, &referral)
	if err != nil {
		return &referral, err
	}
	return &referral, nil
}

// GetReferralFromEmail .....
func (db *DSReferral) GetReferralFromEmail(ctx context.Context, emailID string) ([]contracts.DSReferral, error) {
	returnedReferrals := make([]contracts.DSReferral, 0)
	allReferrals := make([]contracts.DSReferral, 0)

	qP := datastore.NewQuery("ClinicReferrals")
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	if emailID != "" {
		qP = qP.Filter("PatientEmail =", emailID).Filter("IsDirty =", false)
	}
	keysClinics, err := db.client.GetAll(ctx, qP, &returnedReferrals)
	if err != nil || len(keysClinics) <= 0 {
		return nil, fmt.Errorf("no referrals found: %v", err)
	}
	for _, ref := range returnedReferrals {
		if strings.Contains(strings.ToLower(ref.Status.SPStatus), "complete") || strings.Contains(strings.ToLower(ref.Status.SPStatus), "finish") ||
			strings.Contains(strings.ToLower(ref.Status.SPStatus), "close") {
			continue
		}
		allReferrals = append(allReferrals, ref)
	}
	return allReferrals, nil
}

// GetReferralUsingFields .....
func (db *DSReferral) GetReferralUsingFields(ctx context.Context, fromEmail string, pfName string, plName string) ([]contracts.DSReferral, error) {
	returnedReferrals := make([]contracts.DSReferral, 0)
	allReferrals := make([]contracts.DSReferral, 0)

	qP := datastore.NewQuery("ClinicReferrals")
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	if fromEmail != "" {
		qP = qP.Filter("FromEmail =", fromEmail)
	}
	if pfName != "" {
		qP = qP.Filter("PatientFirstName =", pfName)
	}
	if plName != "" {
		qP = qP.Filter("PatientLastName =", plName)
	}
	qP = qP.Filter("IsDirty =", false)
	keysClinics, err := db.client.GetAll(ctx, qP, &returnedReferrals)
	if err != nil || len(keysClinics) <= 0 {
		return nil, fmt.Errorf("no referrals found: %v", err)
	}
	for _, ref := range returnedReferrals {
		if !ref.IsSummary {
			if strings.Contains(strings.ToLower(ref.Status.SPStatus), "complete") || strings.Contains(strings.ToLower(ref.Status.SPStatus), "finish") ||
				strings.Contains(strings.ToLower(ref.Status.SPStatus), "close") {
				continue
			}
		}
		allReferrals = append(allReferrals, ref)
	}
	return allReferrals, nil
}

// ReferralFromPatientPhone .....
func (db *DSReferral) ReferralFromPatientPhone(ctx context.Context, patientPhone string) ([]contracts.DSReferral, error) {
	returnedReferrals := make([]contracts.DSReferral, 0)
	qP := datastore.NewQuery("ClinicReferrals")
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	if patientPhone != "" {
		qP = qP.Filter("PatientPhone =", patientPhone).Filter("IsDirty =", false)
	}
	keysClinics, err := db.client.GetAll(ctx, qP, &returnedReferrals)
	if err != nil || len(keysClinics) <= 0 {
		return nil, fmt.Errorf("no referrals found: %v", err)
	}
	outputRef := make([]contracts.DSReferral, 0)
	for _, ref := range returnedReferrals {
		if strings.Contains(strings.ToLower(ref.Status.SPStatus), "complete") || strings.Contains(strings.ToLower(ref.Status.SPStatus), "finish") ||
			strings.Contains(strings.ToLower(ref.Status.SPStatus), "close") {
			continue
		}
		outputRef = append(outputRef, ref)
	}
	return outputRef, nil
}

// GetAllReferralsGD .....
func (db *DSReferral) GetAllReferralsGD(ctx context.Context, addressID string) ([]contracts.DSReferral, error) {
	returnedReferrals := make([]contracts.DSReferral, 0)
	qP := datastore.NewQuery("ClinicReferrals")
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	if addressID != "" {
		qP = qP.Filter("FromAddressID =", addressID).Filter("IsDirty =", false)
	}
	keysClinics, err := db.client.GetAll(ctx, qP, &returnedReferrals)
	if err != nil || len(keysClinics) <= 0 {
		return returnedReferrals, fmt.Errorf("no referrals found: %v", err)
	}
	return returnedReferrals, nil
}

// GetAllReferralsGDPaginate .....
func (db *DSReferral) GetAllReferralsGDPaginate(ctx context.Context, addressID string, pageSize int, cursor string) ([]contracts.DSReferral, string, error) {
	returnedReferrals := make([]contracts.DSReferral, 0)
	qP := datastore.NewQuery("ClinicReferrals").Limit(pageSize)
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	if cursor != "" {
		cursor, err := datastore.DecodeCursor(cursor)
		if err != nil {
			return returnedReferrals, "", err
		}
		qP = qP.Start(cursor)
	}
	if addressID != "" {
		qP = qP.Filter("FromAddressID =", addressID).Filter("IsDirty =", false)
	}
	iteratorClinics := db.client.Run(ctx, qP)
	for {
		var referral contracts.DSReferral
		_, err := iteratorClinics.Next(&referral)
		if err != nil {
			log.Printf("query ended %q: %v", cursor, err)
			break
		}
		returnedReferrals = append(returnedReferrals, referral)
		// Do something with the Person p
	}
	nextCursor, err := iteratorClinics.Cursor()
	if err != nil {
		return returnedReferrals, "", err
	}
	return returnedReferrals, nextCursor.String(), nil
}

// GetAllTreamentSummaryGD .....
func (db *DSReferral) GetAllTreamentSummaryGD(ctx context.Context, placeID string) ([]contracts.DSReferral, error) {
	returnedReferrals := make([]contracts.DSReferral, 0)
	qP := datastore.NewQuery("ClinicReferrals")
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	if placeID != "" {
		qP = qP.Filter("FromPlaceID =", placeID).Filter("IsDirty =", false)
	}
	keysClinics, err := db.client.GetAll(ctx, qP, &returnedReferrals)
	if err != nil || len(keysClinics) <= 0 {
		return returnedReferrals, fmt.Errorf("no referrals found: %v", err)
	}
	return returnedReferrals, nil
}

// GetAllReferralsSP .....
func (db *DSReferral) GetAllReferralsSP(ctx context.Context, addressID string, clinicName string) ([]contracts.DSReferral, error) {
	returnedReferrals := make([]contracts.DSReferral, 0)
	qP := datastore.NewQuery("ClinicReferrals")
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	if addressID != "" {
		qP = qP.Filter("ToPlaceID =", addressID).Filter("IsDirty =", false)
	}
	_, err := db.client.GetAll(ctx, qP, &returnedReferrals)
	if err != nil {
		return returnedReferrals, fmt.Errorf("no referrals found: %v", err)
	}
	return returnedReferrals, nil
}

// GetAllReferralsSPPaginate .....
func (db *DSReferral) GetAllReferralsSPPaginate(ctx context.Context, addressID string, clinicName string, pageSize int, cursor string) ([]contracts.DSReferral, string, error) {
	returnedReferrals := make([]contracts.DSReferral, 0)
	qP := datastore.NewQuery("ClinicReferrals").Limit(pageSize)
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	if cursor != "" {
		cursor, err := datastore.DecodeCursor(cursor)
		if err != nil {
			return returnedReferrals, "", err
		}
		qP = qP.Start(cursor)
	}
	if addressID != "" {
		qP = qP.Filter("ToPlaceID =", addressID).Filter("IsDirty =", false)
	}
	iteratorClinics := db.client.Run(ctx, qP)
	for {
		var referral contracts.DSReferral
		_, err := iteratorClinics.Next(&referral)
		if err != nil {
			log.Printf("query ended %q: %v", cursor, err)
			break
		}
		returnedReferrals = append(returnedReferrals, referral)
		// Do something with the Person p
	}
	nextCursor, err := iteratorClinics.Cursor()
	if err != nil {
		return returnedReferrals, "", err
	}
	return returnedReferrals, nextCursor.String(), nil
}

// DeleteReferral .....
func (db *DSReferral) DeleteReferral(ctx context.Context, refID string) (*contracts.DSReferral, error) {
	primaryKey := datastore.NameKey("ClinicReferrals", refID, nil)
	if global.Options.DSName != "" {
		primaryKey.Namespace = global.Options.DSName
	}
	var referral contracts.DSReferral
	err := db.client.Delete(ctx, primaryKey)
	if err != nil {
		return nil, err
	}
	return &referral, nil
}

// CreateMessage .....
func (db *DSReferral) CreateMessage(ctx context.Context, referral contracts.DSReferral, comms []contracts.Comment) error {
	primaryKey := datastore.NameKey("ReferralMessages", referral.ReferralID, nil)
	if global.Options.DSName != "" {
		primaryKey.Namespace = global.Options.DSName
	}
	for _, comment := range comms {
		secondarKey := datastore.NameKey("ReferralMessages", comment.MessageID, primaryKey)
		if global.Options.DSName != "" {
			secondarKey.Namespace = global.Options.DSName
		}
		_, err := db.client.Put(ctx, secondarKey, &comment)
		if err != nil {
			return fmt.Errorf("cannot register clinic with sd: %v", err)
		}
	}
	return nil
}

// GetMessagesAll .....
func (db *DSReferral) GetMessagesAll(ctx context.Context, referralID string) ([]contracts.Comment, error) {
	ancKey := datastore.NameKey("ReferralMessages", referralID, nil)
	if global.Options.DSName != "" {
		ancKey.Namespace = global.Options.DSName
	}
	returnedComments := make([]contracts.Comment, 0)
	qP := datastore.NewQuery("ReferralMessages").Ancestor(ancKey)
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	keysClinics, err := db.client.GetAll(ctx, qP, &returnedComments)
	if err != nil || len(keysClinics) <= 0 {
		return nil, fmt.Errorf("no comments found: %v", err)
	}
	return returnedComments, nil
}

// GetMessagesAllWithChannel .....
func (db *DSReferral) GetMessagesAllWithChannel(ctx context.Context, referralID string, channel string) ([]contracts.Comment, error) {
	ancKey := datastore.NameKey("ReferralMessages", referralID, nil)
	if global.Options.DSName != "" {
		ancKey.Namespace = global.Options.DSName
	}
	returnedComments := make([]contracts.Comment, 0)
	qP := datastore.NewQuery("ReferralMessages").Ancestor(ancKey)
	if channel != "" {
		qP = qP.Filter("Channel =", channel)
	}
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	keysClinics, err := db.client.GetAll(ctx, qP, &returnedComments)
	if err != nil || len(keysClinics) <= 0 {
		return nil, fmt.Errorf("no comments found: %v", err)
	}
	return returnedComments, nil
}

// GetOneMessage .....
func (db *DSReferral) GetOneMessage(ctx context.Context, referralID string, messageID string) (*contracts.Comment, error) {
	pKey := datastore.NameKey("ReferralMessages", referralID, nil)
	if global.Options.DSName != "" {
		pKey.Namespace = global.Options.DSName
	}
	mainKey := datastore.NameKey("ReferralMessages", messageID, pKey)
	if global.Options.DSName != "" {
		mainKey.Namespace = global.Options.DSName
	}
	var returnedComments contracts.Comment
	err := db.client.Get(ctx, mainKey, &returnedComments)
	if err != nil {
		return nil, fmt.Errorf("no comments found: %v", err)
	}
	return &returnedComments, nil
}
