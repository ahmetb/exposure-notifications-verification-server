// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package realmadmin_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/exposure-notifications-verification-server/internal/browser"
	"github.com/google/exposure-notifications-verification-server/internal/envstest"
	"github.com/google/exposure-notifications-verification-server/internal/project"
	"github.com/google/exposure-notifications-verification-server/pkg/controller"
	"github.com/google/exposure-notifications-verification-server/pkg/database"

	"github.com/chromedp/chromedp"
)

func TestHandleSettings_SMS(t *testing.T) {
	t.Parallel()

	harness := envstest.NewServer(t, testDatabaseInstance)

	// Get the default realm
	realm, err := harness.Database.FindRealm(1)
	if err != nil {
		t.Fatal(err)
	}

	// Create a user
	admin := &database.User{
		Email:       "admin@example.com",
		Name:        "Admin",
		Realms:      []*database.Realm{realm},
		AdminRealms: []*database.Realm{realm},
	}
	if err := harness.Database.SaveUser(admin, database.System); err != nil {
		t.Fatal(err)
	}

	// Log in the user.
	session, err := harness.LoggedInSession(nil, admin.Email)
	if err != nil {
		t.Fatal(err)
	}

	// Set the current realm.
	controller.StoreSessionRealm(session, realm)

	// Mint a cookie for the session.
	cookie, err := harness.SessionCookie(session)
	if err != nil {
		t.Fatal(err)
	}

	// Create a browser runner.
	browserCtx := browser.New(t)
	taskCtx, done := context.WithTimeout(browserCtx, 30*time.Second)
	defer done()

	var twilioAccountSid string
	var twilioAuthToken string
	var twilioFromNumber string

	if err := chromedp.Run(taskCtx,
		// Pre-authenticate the user.
		browser.SetCookie(cookie),

		// Visit page.
		chromedp.Navigate(`http://`+harness.Server.Addr()+`/realm/settings#sms`),

		// Wait for render.
		chromedp.WaitVisible(`div#sms`, chromedp.ByQuery),

		// Fill out the form.
		chromedp.SetValue(`input#twilio-account-sid`, "accountSid", chromedp.ByQuery),
		chromedp.SetValue(`input#twilio-auth-token`, "authToken", chromedp.ByQuery),
		chromedp.SetValue(`input#twilio-from-number`, "1234567890", chromedp.ByQuery),

		// Click submit.
		chromedp.Click(`input#update-sms`, chromedp.ByQuery),

		// Wait for the page to reload.
		chromedp.WaitVisible(`div#sms`, chromedp.ByQuery),

		chromedp.Value(`input#twilio-account-sid`, &twilioAccountSid, chromedp.ByQuery),
		chromedp.Value(`input#twilio-auth-token`, &twilioAuthToken, chromedp.ByQuery),
		chromedp.Value(`input#twilio-from-number`, &twilioFromNumber, chromedp.ByQuery),
	); err != nil {
		t.Fatal(err)
	}

	// Check form
	if got, want := twilioAccountSid, "accountSid"; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
	if got, want := twilioAuthToken, project.PasswordSentinel; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
	if got, want := twilioFromNumber, "1234567890"; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}

	{
		// Check database
		smsConfig, err := realm.SMSConfig(harness.Database)
		if err != nil {
			t.Fatal(err)
		}
		if smsConfig == nil {
			t.Fatal("expected smsConfig")
		}

		if got, want := smsConfig.TwilioAccountSid, "accountSid"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}
		if got, want := smsConfig.TwilioAuthToken, "authToken"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}
		if got, want := smsConfig.TwilioFromNumber, "1234567890"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}
	}

	// Update
	if err := chromedp.Run(taskCtx,
		// Pre-authenticate the user.
		browser.SetCookie(cookie),

		// Visit page.
		chromedp.Navigate(`http://`+harness.Server.Addr()+`/realm/settings#sms`),

		// Wait for render.
		chromedp.WaitVisible(`div#sms`, chromedp.ByQuery),

		// Fill out the form.
		chromedp.SetValue(`input#twilio-account-sid`, "accountSid-new", chromedp.ByQuery),
		chromedp.SetValue(`input#twilio-from-number`, "1234567890-new", chromedp.ByQuery),

		// Click submit.
		chromedp.Click(`input#update-sms`, chromedp.ByQuery),

		// Wait for the page to reload.
		chromedp.WaitVisible(`div#sms`, chromedp.ByQuery),
	); err != nil {
		t.Fatal(err)
	}

	{
		// Check database again
		smsConfig, err := realm.SMSConfig(harness.Database)
		if err != nil {
			t.Fatal(err)
		}
		if smsConfig == nil {
			t.Fatal("expected smsConfig")
		}

		if got, want := smsConfig.TwilioAccountSid, "accountSid-new"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}
		if got, want := smsConfig.TwilioAuthToken, "authToken"; got != want {
			// should not change
			t.Errorf("expected %q to be %q", got, want)
		}
		if got, want := smsConfig.TwilioFromNumber, "1234567890-new"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}
	}

	// Delete
	if err := chromedp.Run(taskCtx,
		// Pre-authenticate the user.
		browser.SetCookie(cookie),

		// Visit page.
		chromedp.Navigate(`http://`+harness.Server.Addr()+`/realm/settings#sms`),

		// Wait for render.
		chromedp.WaitVisible(`div#sms`, chromedp.ByQuery),

		// Fill out the form.
		chromedp.SetValue(`input#twilio-account-sid`, "", chromedp.ByQuery),
		chromedp.SetValue(`input#twilio-auth-token`, "", chromedp.ByQuery),
		chromedp.SetValue(`input#twilio-from-number`, "", chromedp.ByQuery),

		// Click submit.
		chromedp.Click(`input#update-sms`, chromedp.ByQuery),

		// Wait for the page to reload.
		chromedp.WaitVisible(`div#sms`, chromedp.ByQuery),
	); err != nil {
		t.Fatal(err)
	}

	// Check database again
	if _, err := realm.SMSConfig(harness.Database); !database.IsNotFound(err) {
		t.Fatal("expected smsConfig to be deleted")
	}
}
