package acceptance

import (
	"context"
	"fmt"
	"github.com/eurofurence/reg-attendee-service/docs"
	"github.com/eurofurence/reg-attendee-service/internal/api/v1/admin"
	"github.com/eurofurence/reg-attendee-service/internal/api/v1/attendee"
	"github.com/eurofurence/reg-attendee-service/internal/api/v1/status"
	"github.com/eurofurence/reg-attendee-service/internal/entity"
	"github.com/eurofurence/reg-attendee-service/internal/repository/config"
	"github.com/eurofurence/reg-attendee-service/internal/repository/database"
	"github.com/eurofurence/reg-attendee-service/internal/repository/mailservice"
	"github.com/eurofurence/reg-attendee-service/internal/repository/paymentservice"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"
)

// -------------------------------------------
// acceptance tests for the status subresource
// -------------------------------------------

// -- read status

func TestStatus_AnonDeny(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an existing attendee")
	attendeeLocation, _ := tstRegisterAttendee(t, "stat1-")

	docs.Given("given an unauthenticated user")
	token := tstNoToken()

	docs.When("when they attempt to access the status")
	response := tstPerformGet(attendeeLocation+"/status", token)

	docs.Then("then the request is denied as unauthenticated (401) and the appropriate error is returned")
	tstRequireErrorResponse(t, response, http.StatusUnauthorized, "auth.unauthorized", "you must be logged in for this operation")
}

func TestStatus_UserDenyOther(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given two existing regular users, the second of which has registered")
	token := tstValidUserToken(t, "101")
	location2, _ := tstRegisterAttendee(t, "stat2b-")

	docs.When("when the first user attempts to access somebody else's status")
	response := tstPerformGet(location2+"/status", token)

	docs.Then("then the request is denied as unauthorized (403) and the appropriate error is returned")
	tstRequireErrorResponse(t, response, http.StatusForbidden, "auth.forbidden", "you are not authorized to access this data - the attempt has been logged")
}

func TestStatus_UserAllowSelf(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an existing attendee")
	token := tstValidUserToken(t, "101")
	location1, _ := tstRegisterAttendeeWithToken(t, "stat3-", token)

	docs.When("when they access their own status")
	response := tstPerformGet(location1+"/status", token)

	docs.Then("then the request is successful and status 'new' is returned")
	statusDto := status.StatusDto{}
	tstParseJson(response.body, &statusDto)
	expectedStatus := status.StatusDto{
		Status: "new",
	}
	require.EqualValues(t, expectedStatus, statusDto, "status did not match expected value")
}

func TestStatus_StaffDenyOther(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, true, true))
	defer tstShutdown()

	docs.Given("given two existing users, the first of which is staff")
	token1 := tstValidStaffToken(t, "202")
	location2, _ := tstRegisterAttendee(t, "stat4b-")

	docs.When("when the staffer attempts to access somebody else's status")
	response := tstPerformGet(location2+"/status", token1)

	docs.Then("then the request is denied as unauthorized (403) and the appropriate error is returned")
	tstRequireErrorResponse(t, response, http.StatusForbidden, "auth.forbidden", "you are not authorized to access this data - the attempt has been logged")
}

func TestStatus_StaffAllowSelf(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, true, true))
	defer tstShutdown()

	docs.Given("given an existing attendee who is staff")
	token := tstValidStaffToken(t, "202")
	location1, _ := tstRegisterAttendeeWithToken(t, "stat5-", token)

	docs.When("when they access their own status")
	response := tstPerformGet(location1+"/status", token)

	docs.Then("then the request is successful and status 'new' is returned")
	statusDto := status.StatusDto{}
	tstParseJson(response.body, &statusDto)
	expectedStatus := status.StatusDto{
		Status: "new",
	}
	require.EqualValues(t, expectedStatus, statusDto, "status did not match expected value")
}

func TestStatus_AdminOk(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an existing attendee")
	location1, _ := tstRegisterAttendee(t, "stat6-")

	docs.Given("given a logged in admin")
	token := tstValidAdminToken(t)

	docs.When("when they access the status for any attendee")
	response := tstPerformGet(location1+"/status", token)

	docs.Then("then the request is successful and the default status is returned")
	require.Equal(t, http.StatusOK, response.status, "unexpected http response status")
	tstRequireAttendeeStatus(t, "new", response.body)
}

func TestStatus_InvalidId(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given a logged in admin")
	token := tstValidAdminToken(t)

	docs.When("when they try to access the status for an attendee with an invalid id")
	response := tstPerformGet("/api/rest/v1/attendees/panther/status", token)

	docs.Then("then the request fails and the appropriate error is returned")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "attendee.id.invalid", url.Values{})
}

func TestStatus_Nonexistent(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given a logged in admin")
	token := tstValidAdminToken(t)

	docs.When("when they try to access the status for an attendee that does not exist")
	response := tstPerformGet("/api/rest/v1/attendees/42/status", token)

	docs.Then("then the request fails and the appropriate error is returned")
	tstRequireErrorResponse(t, response, http.StatusNotFound, "attendee.id.notfound", url.Values{})
}

// -- status history

func TestStatusHistory_AnonDeny(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an existing attendee")
	location1, _ := tstRegisterAttendee(t, "stat20-")

	docs.Given("given an unauthenticated user")
	token := tstNoToken()

	docs.When("when they attempt to access the status history")
	response := tstPerformGet(location1+"/status-history", token)

	docs.Then("then the request is denied as unauthenticated (401) and the appropriate error is returned")
	tstRequireErrorResponse(t, response, http.StatusUnauthorized, "auth.unauthorized", "you must be logged in for this operation")
}

func TestStatusHistory_SelfDeny(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an existing attendee")
	token := tstValidUserToken(t, "101")
	location1, _ := tstRegisterAttendeeWithToken(t, "stat21-", token)

	docs.When("when they attempt to access their own status history")
	response := tstPerformGet(location1+"/status-history", token)

	docs.Then("then the request is denied as unauthorized (403) and the appropriate error is returned (status history is admin only)")
	tstRequireErrorResponse(t, response, http.StatusForbidden, "auth.forbidden", "you are not authorized for this operation - the attempt has been logged")
}

func TestStatusHistory_UserDeny(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an existing attendee")
	location1, attendee1 := tstRegisterAttendee(t, "stat21-")

	docs.Given("given a regular authenticated attendee")
	token := tstValidUserToken(t, attendee1.Id)

	docs.When("when they attempt to access somebody else's status history")
	response := tstPerformGet(location1+"/status-history", token)

	docs.Then("then the request is denied as unauthorized (403) and the appropriate error is returned")
	tstRequireErrorResponse(t, response, http.StatusForbidden, "auth.forbidden", "you are not authorized for this operation - the attempt has been logged")
}

func TestStatusHistory_StaffDeny(t *testing.T) {
	docs.Given("given the configuration for staff registration")
	tstSetup(tstConfigFile(false, true, true))
	defer tstShutdown()

	docs.Given("given an authenticated staffer who has made a valid registration")
	token := tstValidStaffToken(t, "202")
	location1, _ := tstRegisterAttendeeWithToken(t, "stat22-", token)

	docs.When("when they attempt to access their own (or somebody else's) status history")
	response := tstPerformGet(location1+"/status-history", token)

	docs.Then("then the request is denied as unauthorized (403) and the appropriate error is returned")
	tstRequireErrorResponse(t, response, http.StatusForbidden, "auth.forbidden", "you are not authorized for this operation - the attempt has been logged")
}

func TestStatusHistory_AdminOk(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an existing attendee right after registration")
	location1, attendee1 := tstRegisterAttendee(t, "stat23-")

	docs.Given("given a logged in admin")
	token := tstValidAdminToken(t)

	docs.When("when they access the status history for any attendee")
	response := tstPerformGet(location1+"/status-history", token)

	docs.Then("then the request is successful and the default status history is returned")
	require.Equal(t, http.StatusOK, response.status, "unexpected http response status")
	statusHistoryDto := status.StatusHistoryDto{}
	tstParseJson(response.body, &statusHistoryDto)

	require.Equal(t, 1, len(statusHistoryDto.StatusHistory))
	expectedStatusHistory := status.StatusHistoryDto{
		Id: attendee1.Id,
		StatusHistory: []status.StatusChangeDto{{
			Timestamp: statusHistoryDto.StatusHistory[0].Timestamp,
			Status:    "new",
			Comment:   "registration",
		}},
	}
	require.EqualValues(t, expectedStatusHistory, statusHistoryDto, "status history did not match expected value")
}

func TestStatusHistory_InvalidId(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given a logged in admin")
	token := tstValidAdminToken(t)

	docs.When("when they try to access the status history for an attendee with an invalid id")
	response := tstPerformGet("/api/rest/v1/attendees/lynx/status-history", token)

	docs.Then("then the request fails and the appropriate error is returned")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "attendee.id.invalid", url.Values{})
}

func TestStatusHistory_Nonexistent(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given a logged in admin")
	token := tstValidAdminToken(t)

	docs.When("when they try to access the status history for an attendee that does not exist")
	response := tstPerformGet("/api/rest/v1/attendees/42/status-history", token)

	docs.Then("then the request fails and the appropriate error is returned")
	tstRequireErrorResponse(t, response, http.StatusNotFound, "attendee.id.notfound", url.Values{})
}

// --- status changes ---

func TestStatusChange_InvalidId(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.When("when an admin attempts a status change for an invalid attendee id")
	body := status.StatusChangeDto{
		Status:  "approved",
		Comment: "stat40",
	}
	response := tstPerformPost("/api/rest/v1/attendees/tigress/status", tstRenderJson(body), tstValidAdminToken(t))

	docs.Then("then the request fails and the appropriate error is returned")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "attendee.id.invalid", "")
}

func TestStatusChange_Nonexistant(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.When("when an admin attempts a status change for an attendee that does not exist")
	body := status.StatusChangeDto{
		Status:  "approved",
		Comment: "stat41",
	}
	response := tstPerformPost("/api/rest/v1/attendees/444/status", tstRenderJson(body), tstValidAdminToken(t))

	docs.Then("then the request fails and the appropriate error is returned")
	tstRequireErrorResponse(t, response, http.StatusNotFound, "attendee.id.notfound", "")

	docs.Then("and no dues or payment changes have been recorded")
	require.Empty(t, paymentMock.Recording())

	docs.Then("and no email messages have been sent")
	require.Empty(t, mailMock.Recording())
}

func TestStatusChange_InvalidBodySyntax(t *testing.T) {
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an attendee in status approved")
	loc, _ := tstRegisterAttendeeAndTransitionToStatus(t, "stat42", "approved")

	docs.When("when an admin prematurely tries to change their status but sends a syntactically invalid request body")
	response := tstPerformPost(loc+"/status", "{{-}}}}", tstValidAdminToken(t))

	docs.Then("then the request fails and the appropriate error is returned")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "status.parse.error", url.Values{})

	docs.Then("and the status is unchanged")
	tstVerifyStatus(t, loc, "approved")

	docs.Then("and no dues or payment changes have been recorded")
	require.Empty(t, paymentMock.Recording())

	docs.Then("and no email messages have been sent")
	require.Empty(t, mailMock.Recording())
}

func TestStatusChange_InvalidBodyValues(t *testing.T) {
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an attendee in status approved")
	loc, _ := tstRegisterAttendeeAndTransitionToStatus(t, "stat43", "approved")

	docs.When("when an admin tries to change the status to an invalid value")
	body := status.StatusChangeDto{
		Status:  "fluffy",
		Comment: "why isn't there a status fluffy?",
	}
	response := tstPerformPost(loc+"/status", tstRenderJson(body), tstValidAdminToken(t))

	docs.Then("then the request fails and the appropriate error is returned")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "status.data.invalid", url.Values{
		"status": []string{"status must be one of new,approved,partially paid,paid,checked in,cancelled,deleted"},
	})

	docs.Then("and the status is unchanged")
	tstVerifyStatus(t, loc, "approved")

	docs.Then("and no dues or payment changes have been recorded")
	require.Empty(t, paymentMock.Recording())

	docs.Then("and no email messages have been sent")
	require.Empty(t, mailMock.Recording())
}

// - anonymous is always denied -

func TestStatusChange_Anonymous_Any_Any(t *testing.T) {
	for o, oldStatus := range config.AllowedStatusValues() {
		for n, newStatus := range config.AllowedStatusValues() {
			testname := fmt.Sprintf("TestStatusChange_Anonymous_%s_%s", oldStatus, newStatus)
			t.Run(testname, func(t *testing.T) {
				tstStatusChange_Anonymous_Deny(t, fmt.Sprintf("st%danon%d-", o, n), oldStatus, newStatus)
			})
		}
	}
}

// - other (without regdesk permission) is always denied -

func TestStatusChange_Other_Any_Any(t *testing.T) {
	for o, oldStatus := range config.AllowedStatusValues() {
		for n, newStatus := range config.AllowedStatusValues() {
			testname := fmt.Sprintf("TestStatusChange_Other_%s_%s", oldStatus, newStatus)
			t.Run(testname, func(t *testing.T) {
				tstStatusChange_Other_Deny(t, fmt.Sprintf("st%dother%d-", o, n), oldStatus, newStatus)
			})
		}
	}
}

// - staff (without regdesk permission they are no different from regular attendees) is always denied for others -

func TestStatusChange_Staff_Other_Any_Any(t *testing.T) {
	for o, oldStatus := range config.AllowedStatusValues() {
		for n, newStatus := range config.AllowedStatusValues() {
			testname := fmt.Sprintf("TestStatusChange_Staff_%s_%s", oldStatus, newStatus)
			t.Run(testname, func(t *testing.T) {
				tstStatusChange_Staff_Other_Deny(t, fmt.Sprintf("st%dstaff%d-", o, n), oldStatus, newStatus)
			})
		}
	}
}

// - self can do self cancellation from new and approved, but nothing else -
// (note that received payments come in as admin requests either from the payment service or from an admin, so those aren't self reported)

func TestStatusChange_Self_New_Cancelled(t *testing.T) {
	testcase := "st0self6-"
	tstStatusChange_Self_Allow(t, testcase,
		"new", "cancelled",
		[]paymentservice.Transaction{},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "cancelled")},
	)
}

func TestStatusChange_Self_Approved_Cancelled(t *testing.T) {
	testcase := "st1self6-"
	tstStatusChange_Self_Allow(t, testcase,
		"approved", "cancelled",
		[]paymentservice.Transaction{tstValidAttendeeDues(-25500, "void unpaid dues on cancel")},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "cancelled")},
	)
}

// TODO refund logic by self cancellation date

func TestStatusChange_Self_Any_Any(t *testing.T) {
	for o, oldStatus := range config.AllowedStatusValues() {
		for n, newStatus := range config.AllowedStatusValues() {
			if (oldStatus == "new" || oldStatus == "approved") && newStatus == "cancelled" {
				// see individual test cases above
			} else {
				testname := fmt.Sprintf("TestStatusChange_Self_%s_%s", oldStatus, newStatus)
				t.Run(testname, func(t *testing.T) {
					tstStatusChange_Self_Deny(t, fmt.Sprintf("st%dself%d-", o, n), oldStatus, newStatus)
				})
			}
		}
	}
}

// TODO test self cancellation unavailable because there were payments, even if refunded

// - an attendee with regdesk permission can check fully paid people in, but can do nothing else -

func TestStatusChange_Regdesk_Paid_CheckedIn(t *testing.T) {
	testcase := "st3regdsk4-"
	tstStatusChange_Regdesk_Allow(t, testcase,
		"paid", "checked in",
		[]paymentservice.Transaction{},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "checked in")},
	)
}

func TestStatusChange_Regdesk_NotCompletelyPaid_CheckedIn(t *testing.T) {
	tstStatusChange_Regdesk_Unavailable(t, "st3regdsk4a-",
		"paid", "checked in",
		[]paymentservice.Transaction{tstCreateTransaction(1, paymentservice.Payment, -42)}, // slight underpayment
		"status.unpaid.dues", "payment amount not sufficient",
	)
}

func TestStatusChange_Regdesk_Any_Any(t *testing.T) {
	for o, oldStatus := range config.AllowedStatusValues() {
		for n, newStatus := range config.AllowedStatusValues() {
			if (oldStatus == "new" || oldStatus == "approved") && newStatus == "cancelled" {
				// see normal user test cases above - everyone may self-cancel here
			} else if oldStatus == "paid" && newStatus == "checked in" {
				// see individual test case above
			} else {
				testname := fmt.Sprintf("TestStatusChange_Regdesk_%s_%s", oldStatus, newStatus)
				t.Run(testname, func(t *testing.T) {
					tstStatusChange_Regdesk_Deny(t, fmt.Sprintf("st%dregdsk%d-", o, n), oldStatus, newStatus)
				})
			}
		}
	}
}

// - admins can make any available status change, so this tests availability conditions, mails sent and payment bookings
//   in all these cases -

func TestStatusChange_Admin_Same_Same(t *testing.T) {
	for b, bothStatus := range config.AllowedStatusValues() {
		testname := fmt.Sprintf("TestStatusChange_Admin_%s_%s", bothStatus, bothStatus)
		t.Run(testname, func(t *testing.T) {
			tstStatusChange_Admin_Unavailable(t, fmt.Sprintf("st%dadm%d-", b, b),
				bothStatus, bothStatus,
				nil,
				"status.unchanged.invalid", "old and new status are the same")
		})
	}
}

func TestStatusChange_Admin_New_Approved(t *testing.T) {
	testcase := "st0adm1-"
	tstStatusChange_Admin_Allow(t, testcase,
		"new", "approved",
		nil,
		[]paymentservice.Transaction{tstValidAttendeeDues(25500, "dues adjustment due to change in status or selected packages")},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "approved")},
	)
}

func TestStatusChange_Admin_New_Cancelled(t *testing.T) {
	testcase := "st0adm5-"
	tstStatusChange_Admin_Allow(t, testcase,
		"new", "cancelled",
		nil,
		[]paymentservice.Transaction{},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "cancelled")},
	)
}

func TestStatusChange_Admin_New_Deleted(t *testing.T) {
	testcase := "st0adm6-"
	tstStatusChange_Admin_Allow(t, testcase,
		"new", "deleted",
		nil,
		[]paymentservice.Transaction{},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "deleted")},
	)
}

func TestStatusChange_Admin_New_Any(t *testing.T) {
	for n, targetStatus := range config.AllowedStatusValues() {
		if targetStatus == "partially paid" || targetStatus == "paid" || targetStatus == "checked in" {
			testname := fmt.Sprintf("TestStatusChange_Admin_%s_%s", "new", targetStatus)
			t.Run(testname, func(t *testing.T) {
				tstStatusChange_Admin_Unavailable(t, fmt.Sprintf("st%dadm%d-", 0, n),
					"new", targetStatus,
					nil,
					"status.use.approved", "please change status to approved, this will automatically advance to (partially) paid as appropriate")
			})

		}
	}
}

func TestStatusChange_Admin_Approved_New(t *testing.T) {
	testcase := "st1adm0-"
	tstStatusChange_Admin_Allow(t, testcase,
		"approved", "new",
		nil,
		[]paymentservice.Transaction{tstValidAttendeeDues(-25500, "remove dues balance - status changed to new")},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "new")},
	)
}

func TestStatusChange_Admin_Approved_PartiallyPaid(t *testing.T) {
	testcase := "st1adm2-"
	tstStatusChange_Admin_Allow(t, testcase,
		"approved", "partially paid",
		[]paymentservice.Transaction{tstCreateTransaction(1, paymentservice.Payment, 2040)},
		[]paymentservice.Transaction{},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "partially paid")},
	)
}

func TestStatusChange_Admin_Approved_Paid_WithGraceAmount(t *testing.T) {
	testcase := "st1adm3-"
	tstStatusChange_Admin_Allow(t, testcase,
		"approved", "paid",
		[]paymentservice.Transaction{tstCreateTransaction(1, paymentservice.Payment, 25400)},
		[]paymentservice.Transaction{},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "paid")},
	)
}

func TestStatusChange_Admin_Approved_CheckedIn(t *testing.T) {
	testcase := "st1adm4-"
	tstStatusChange_Admin_Allow(t, testcase,
		"approved", "checked in",
		[]paymentservice.Transaction{tstCreateTransaction(1, paymentservice.Payment, 25500)},
		[]paymentservice.Transaction{},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "checked in")},
	)
}

func TestStatusChange_Admin_Approved_Cancelled(t *testing.T) {
	testcase := "st1adm5-"
	tstStatusChange_Admin_Allow(t, testcase,
		"approved", "cancelled",
		nil,
		[]paymentservice.Transaction{tstValidAttendeeDues(-25500, "void unpaid dues on cancel")},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "cancelled")},
	)
}

func TestStatusChange_Admin_Approved_Deleted(t *testing.T) {
	testcase := "st1adm6-"
	tstStatusChange_Admin_Allow(t, testcase,
		"approved", "deleted",
		nil,
		[]paymentservice.Transaction{tstValidAttendeeDues(-25500, "remove dues balance - status changed to deleted")},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "deleted")},
	)
}

func TestStatusChange_Admin_PartiallyPaid_New(t *testing.T) {
	testcase := "st2adm0-"
	tstStatusChange_Admin_Unavailable(t, testcase,
		"partially paid", "new",
		nil,
		"status.has.paid", "there is a non-zero payment balance, please use partially paid, or refund")
}

func TestStatusChange_Admin_PartiallyPaid_Approved(t *testing.T) {
	testcase := "st2adm1-"
	tstStatusChange_Admin_Unavailable(t, testcase,
		"partially paid", "approved",
		nil,
		"status.has.paid", "there is a non-zero payment balance, please use partially paid, or refund")
}

func TestStatusChange_Admin_PartiallyPaid_Approved_OkAfterRefund(t *testing.T) {
	testcase := "st2adm1r-"
	tstStatusChange_Admin_Allow(t, testcase,
		"partially paid", "approved",
		[]paymentservice.Transaction{tstCreateTransaction(1, paymentservice.Payment, -15500)},
		[]paymentservice.Transaction{},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "approved")},
	)
}

func TestStatusChange_Admin_PartiallyPaid_Paid(t *testing.T) {
	testcase := "st2adm3-"
	tstStatusChange_Admin_Allow(t, testcase,
		"partially paid", "paid",
		[]paymentservice.Transaction{tstCreateTransaction(1, paymentservice.Payment, 10000)},
		[]paymentservice.Transaction{},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "paid")},
	)
}

func TestStatusChange_Admin_PartiallyPaid_CheckedIn(t *testing.T) {
	testcase := "st2adm4-"
	tstStatusChange_Admin_Allow(t, testcase,
		"partially paid", "checked in",
		[]paymentservice.Transaction{tstCreateTransaction(1, paymentservice.Payment, 10000)},
		[]paymentservice.Transaction{},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "checked in")},
	)
}

func TestStatusChange_Admin_PartiallyPaid_Cancelled(t *testing.T) {
	testcase := "st2adm5-"
	tstStatusChange_Admin_Allow(t, testcase,
		"partially paid", "cancelled",
		nil,
		[]paymentservice.Transaction{tstValidAttendeeDues(-10000, "void unpaid dues on cancel")},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "cancelled")},
	)
}

func TestStatusChange_Admin_PartiallyPaid_Deleted(t *testing.T) {
	testcase := "st2adm6-"
	tstStatusChange_Admin_Unavailable(t, testcase,
		"partially paid", "deleted",
		nil,
		"status.cannot.delete", "cannot delete attendee for legal reasons (there were payments or invoices)")
}

func TestStatusChange_Admin_Paid_New(t *testing.T) {
	testcase := "st3adm0-"
	tstStatusChange_Admin_Unavailable(t, testcase,
		"paid", "new",
		nil,
		"status.has.paid", "there is a non-zero payment balance, please use partially paid, or refund")
}

func TestStatusChange_Admin_Paid_New_OkAfterRefund(t *testing.T) {
	testcase := "st3adm0r-"
	tstStatusChange_Admin_Allow(t, testcase,
		"paid", "new",
		[]paymentservice.Transaction{tstCreateTransaction(1, paymentservice.Payment, -25500)},
		[]paymentservice.Transaction{tstCreateMatcherTransaction(1, paymentservice.Due, -25500, "remove dues balance - status changed to new")},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "new")},
	)
}

func TestStatusChange_Admin_Paid_Approved(t *testing.T) {
	testcase := "st3adm1-"
	tstStatusChange_Admin_Unavailable(t, testcase,
		"paid", "approved",
		nil,
		"status.has.paid", "there is a non-zero payment balance, please use partially paid, or refund")
}

func TestStatusChange_Admin_Paid_Approved_OkAfterRefund(t *testing.T) {
	testcase := "st3adm1r-"
	tstStatusChange_Admin_Allow(t, testcase,
		"paid", "approved",
		[]paymentservice.Transaction{tstCreateTransaction(1, paymentservice.Payment, -25500)},
		[]paymentservice.Transaction{},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "approved")},
	)
}

func TestStatusChange_Admin_Paid_PartiallyPaid(t *testing.T) {
	testcase := "st3adm2-"
	tstStatusChange_Admin_Allow(t, testcase,
		"paid", "partially paid",
		[]paymentservice.Transaction{tstCreateTransaction(1, paymentservice.Payment, -10000)},
		[]paymentservice.Transaction{},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "partially paid")},
	)
}

func TestStatusChange_Admin_Paid_CheckedIn_HasNoGraceAmount(t *testing.T) {
	testcase := "st3adm4u-"
	tstStatusChange_Admin_Unavailable(t, testcase,
		"paid", "checked in",
		[]paymentservice.Transaction{tstCreateTransaction(1, paymentservice.Payment, -30)},
		"status.unpaid.dues", "payment amount not sufficient")
}

func TestStatusChange_Admin_Paid_CheckedIn(t *testing.T) {
	testcase := "st3adm4-"
	tstStatusChange_Admin_Allow(t, testcase,
		"paid", "checked in",
		[]paymentservice.Transaction{},
		[]paymentservice.Transaction{},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "checked in")},
	)
}

func TestStatusChange_Admin_Paid_Cancelled(t *testing.T) {
	testcase := "st3adm5-"
	tstStatusChange_Admin_Allow(t, testcase,
		"paid", "cancelled",
		nil,
		[]paymentservice.Transaction{},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "cancelled")},
	)
}

func TestStatusChange_Admin_Paid_Deleted(t *testing.T) {
	testcase := "st3adm6-"
	tstStatusChange_Admin_Unavailable(t, testcase,
		"paid", "deleted",
		nil,
		"status.cannot.delete", "cannot delete attendee for legal reasons (there were payments or invoices)")
}

func TestStatusChange_Admin_CheckedIn_New(t *testing.T) {
	testcase := "st4adm0-"
	tstStatusChange_Admin_Unavailable(t, testcase,
		"checked in", "new",
		nil,
		"status.has.paid", "there is a non-zero payment balance, please use partially paid, or refund")
}

func TestStatusChange_Admin_CheckedIn_New_OkAfterRefund(t *testing.T) {
	testcase := "st4adm0r-"
	tstStatusChange_Admin_Allow(t, testcase,
		"checked in", "new",
		[]paymentservice.Transaction{tstCreateTransaction(1, paymentservice.Payment, -25500)},
		[]paymentservice.Transaction{tstCreateMatcherTransaction(1, paymentservice.Due, -25500, "remove dues balance - status changed to new")},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "new")},
	)
}

func TestStatusChange_Admin_CheckedIn_Approved(t *testing.T) {
	testcase := "st4adm1-"
	tstStatusChange_Admin_Unavailable(t, testcase,
		"checked in", "approved",
		nil,
		"status.has.paid", "there is a non-zero payment balance, please use partially paid, or refund")
}

func TestStatusChange_Admin_CheckedIn_Approved_OkAfterRefund(t *testing.T) {
	testcase := "st4adm1r-"
	tstStatusChange_Admin_Allow(t, testcase,
		"checked in", "approved",
		[]paymentservice.Transaction{tstCreateTransaction(1, paymentservice.Payment, -25500)},
		[]paymentservice.Transaction{},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "approved")},
	)
}

func TestStatusChange_Admin_CheckedIn_PartiallyPaid(t *testing.T) {
	testcase := "st4adm2-"
	tstStatusChange_Admin_Allow(t, testcase,
		"checked in", "partially paid",
		[]paymentservice.Transaction{tstCreateTransaction(1, paymentservice.Payment, -10000)},
		[]paymentservice.Transaction{},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "partially paid")},
	)
}

func TestStatusChange_Admin_CheckedIn_Paid(t *testing.T) {
	testcase := "st4adm3-"
	tstStatusChange_Admin_Allow(t, testcase,
		"checked in", "paid",
		[]paymentservice.Transaction{},
		[]paymentservice.Transaction{},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "paid")},
	)
}

func TestStatusChange_Admin_CheckedIn_Cancelled(t *testing.T) {
	testcase := "st4adm5-"
	tstStatusChange_Admin_Allow(t, testcase,
		"checked in", "cancelled",
		nil,
		[]paymentservice.Transaction{},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "cancelled")},
	)
}

func TestStatusChange_Admin_CheckedIn_Deleted(t *testing.T) {
	testcase := "st4adm6-"
	tstStatusChange_Admin_Unavailable(t, testcase,
		"checked in", "deleted",
		nil,
		"status.cannot.delete", "cannot delete attendee for legal reasons (there were payments or invoices)")
}

func TestStatusChange_Admin_Cancelled_New(t *testing.T) {
	testcase := "st5adm0-"
	tstStatusChange_Admin_Unavailable(t, testcase,
		"cancelled", "new",
		nil,
		"status.has.paid", "there is a non-zero payment balance, please use partially paid, or refund")
}

func TestStatusChange_Admin_Cancelled_New_OkAfterRefund(t *testing.T) {
	testcase := "st5adm0r-"
	tstStatusChange_Admin_Allow(t, testcase,
		"cancelled", "new",
		[]paymentservice.Transaction{tstCreateTransaction(1, paymentservice.Payment, -25500)},
		[]paymentservice.Transaction{tstCreateMatcherTransaction(1, paymentservice.Due, -25500, "remove dues balance - status changed to new")},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "new")},
	)
}

func TestStatusChange_Admin_Cancelled_Approved(t *testing.T) {
	testcase := "st5adm1-"
	tstStatusChange_Admin_Unavailable(t, testcase,
		"cancelled", "approved",
		nil,
		"status.has.paid", "there is a non-zero payment balance, please use partially paid, or refund")
}

func TestStatusChange_Admin_Cancelled_Approved_OkAfterRefund(t *testing.T) {
	testcase := "st5adm1r-"
	tstStatusChange_Admin_Allow(t, testcase,
		"cancelled", "approved",
		[]paymentservice.Transaction{tstCreateTransaction(1, paymentservice.Payment, -25500)},
		[]paymentservice.Transaction{},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "approved")},
	)
}

func TestStatusChange_Admin_Cancelled_PartiallyPaid(t *testing.T) {
	// you cannot directly go back, since there may have been flag, package changes while cancelled which are not reflected in dues
	testcase := "st5adm2-"
	tstStatusChange_Admin_Unavailable(t, testcase,
		"cancelled", "partially paid",
		[]paymentservice.Transaction{tstCreateTransaction(1, paymentservice.Payment, -10000)},
		"status.use.approved", "please change status to approved, this will automatically advance to (partially) paid as appropriate")
}

func TestStatusChange_Admin_Cancelled_Paid(t *testing.T) {
	// you cannot directly go back, since there may have been flag, package changes while cancelled which are not reflected in dues
	testcase := "st5adm3-"
	tstStatusChange_Admin_Unavailable(t, testcase,
		"cancelled", "paid",
		nil,
		"status.use.approved", "please change status to approved, this will automatically advance to (partially) paid as appropriate")
}

func TestStatusChange_Admin_Cancelled_CheckedIn(t *testing.T) {
	// you cannot directly go back, since there may have been flag, package changes while cancelled which are not reflected in dues
	testcase := "st5adm4-"
	tstStatusChange_Admin_Unavailable(t, testcase,
		"cancelled", "checked in",
		nil,
		"status.use.approved", "please change status to approved, this will automatically advance to (partially) paid as appropriate")
}

func TestStatusChange_Admin_Cancelled_Deleted(t *testing.T) {
	testcase := "st5adm6-"
	tstStatusChange_Admin_Unavailable(t, testcase,
		"cancelled", "deleted",
		nil,
		"status.cannot.delete", "cannot delete attendee for legal reasons (there were payments or invoices)")
}

func TestStatusChange_Admin_Deleted_New(t *testing.T) {
	testcase := "st6adm0-"
	tstStatusChange_Admin_Allow(t, testcase,
		"deleted", "new",
		nil,
		[]paymentservice.Transaction{},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "new")},
	)
}

func TestStatusChange_Admin_Deleted_Approved(t *testing.T) {
	testcase := "st6adm1-"
	tstStatusChange_Admin_Allow(t, testcase,
		"deleted", "approved",
		nil,
		[]paymentservice.Transaction{tstValidAttendeeDues(25500, "dues adjustment due to change in status or selected packages")},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "approved")},
	)
}

func TestStatusChange_Admin_Deleted_PartiallyPaid(t *testing.T) {
	// you cannot directly go back, since there may have been flag, package changes while cancelled which are not reflected in dues
	testcase := "st6adm2-"
	tstStatusChange_Admin_Unavailable(t, testcase,
		"deleted", "partially paid",
		[]paymentservice.Transaction{tstCreateTransaction(1, paymentservice.Payment, -10000)},
		"status.use.approved", "please change status to approved, this will automatically advance to (partially) paid as appropriate")
}

func TestStatusChange_Admin_Deleted_Paid(t *testing.T) {
	// you cannot directly go back, since there may have been flag, package changes while cancelled which are not reflected in dues
	testcase := "st6adm3-"
	tstStatusChange_Admin_Unavailable(t, testcase,
		"deleted", "paid",
		nil,
		"status.use.approved", "please change status to approved, this will automatically advance to (partially) paid as appropriate")
}

func TestStatusChange_Admin_Deleted_CheckedIn(t *testing.T) {
	// you cannot directly go back, since there may have been flag, package changes while cancelled which are not reflected in dues
	testcase := "st6adm4-"
	tstStatusChange_Admin_Unavailable(t, testcase,
		"deleted", "checked in",
		nil,
		"status.use.approved", "please change status to approved, this will automatically advance to (partially) paid as appropriate")
}

func TestStatusChange_Admin_Deleted_Cancelled(t *testing.T) {
	testcase := "st6adm5-"
	tstStatusChange_Admin_Allow(t, testcase,
		"deleted", "cancelled",
		nil,
		[]paymentservice.Transaction{},
		[]mailservice.TemplateRequestDto{tstNewStatusMail(testcase, "cancelled")},
	)
}

// ...

// TODO transitions to new or deleted do not get emails

// TODO transition to cancelled and deleted with more complicated dues / payment histories

// TODO ban check

// TODO guest handling

// --- detail implementations for the status change tests ---

func tstStatusChange_Anonymous_Deny(t *testing.T, testcase string, oldStatus string, newStatus string) {
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an attendee in status " + oldStatus)
	loc, _ := tstRegisterAttendeeAndTransitionToStatus(t, testcase, oldStatus)

	docs.When("when an anonymous user tries to change the status to " + newStatus)
	body := status.StatusChangeDto{
		Status:  newStatus,
		Comment: testcase,
	}
	response := tstPerformPost(loc+"/status", tstRenderJson(body), tstNoToken())

	docs.Then("then the request is denied as unauthenticated (401) and the appropriate error is returned")
	tstRequireErrorResponse(t, response, http.StatusUnauthorized, "auth.unauthorized", "you must be logged in for this operation")

	docs.Then("and the status is unchanged")
	tstVerifyStatus(t, loc, oldStatus)

	docs.Then("and no dues or payment changes have been recorded")
	require.Empty(t, paymentMock.Recording())

	docs.Then("and no email messages have been sent")
	require.Empty(t, mailMock.Recording())
}

func tstStatusChange_Self_Deny(t *testing.T, testcase string, oldStatus string, newStatus string) {
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an attendee in status " + oldStatus)
	loc, att := tstRegisterAttendeeAndTransitionToStatus(t, testcase, oldStatus)

	docs.When("when they try to change the status to " + newStatus)
	body := status.StatusChangeDto{
		Status:  newStatus,
		Comment: testcase,
	}
	response := tstPerformPost(loc+"/status", tstRenderJson(body), tstValidUserToken(t, att.Id))

	docs.Then("then the request is denied as unauthorized (403) and the appropriate error is returned")
	tstRequireErrorResponse(t, response, http.StatusForbidden, "auth.forbidden", "you are not allowed to make this status transition - the attempt has been logged")

	docs.Then("and the status is unchanged")
	tstVerifyStatus(t, loc, oldStatus)

	docs.Then("and no dues or payment changes have been recorded")
	require.Empty(t, paymentMock.Recording())

	docs.Then("and no email messages have been sent")
	require.Empty(t, mailMock.Recording())
}

func tstStatusChange_Self_Unavailable(t *testing.T, testcase string, oldStatus string, newStatus string) {
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an attendee in status " + oldStatus)
	loc, att := tstRegisterAttendeeAndTransitionToStatus(t, testcase, oldStatus)

	docs.When("when they prematurely try to change their own status to " + newStatus)
	body := status.StatusChangeDto{
		Status:  newStatus,
		Comment: testcase,
	}
	response := tstPerformPost(loc+"/status", tstRenderJson(body), tstValidStaffToken(t, att.Id))

	docs.Then("then the request fails as conflict (409) and the appropriate error is returned")
	tstRequireErrorResponse(t, response, http.StatusConflict, "", "")

	docs.Then("and the status is unchanged")
	tstVerifyStatus(t, loc, oldStatus)

	docs.Then("and no dues or payment changes have been recorded")
	require.Empty(t, paymentMock.Recording())

	docs.Then("and no email messages have been sent")
	require.Empty(t, mailMock.Recording())
}

func tstStatusChange_Self_Allow(t *testing.T, testcase string,
	oldStatus string, newStatus string,
	expectedTransactions []paymentservice.Transaction,
	expectedMailRequests []mailservice.TemplateRequestDto,
) {
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an attendee in status " + oldStatus)
	token := tstValidStaffToken(t, "1")
	loc, _ := tstRegisterAttendeeAndTransitionToStatus(t, testcase, oldStatus)

	docs.When("when they change their own status to " + newStatus)
	body := status.StatusChangeDto{
		Status:  newStatus,
		Comment: testcase,
	}
	response := tstPerformPost(loc+"/status", tstRenderJson(body), token)

	docs.Then("then the request is successful and the status change has been done")
	require.Equal(t, http.StatusNoContent, response.status)
	tstVerifyStatus(t, loc, newStatus)

	docs.Then("and the appropriate dues were booked in the payment service")
	require.Equal(t, len(expectedTransactions), len(paymentMock.Recording()))
	for i, expected := range expectedTransactions {
		actual := paymentMock.Recording()[i]
		expected.DueDate = actual.DueDate // TODO remove when due date logic implemented
		require.EqualValues(t, expected, actual)
	}

	docs.Then("and the appropriate email messages were sent via the mail service")
	require.Equal(t, len(expectedMailRequests), len(mailMock.Recording()))
	for i, expected := range expectedMailRequests {
		actual := mailMock.Recording()[i]
		require.Contains(t, actual.Email, expected.Email)
		actual.Email = expected.Email
		require.EqualValues(t, expected, actual)
	}
}

func tstStatusChange_Other_Deny(t *testing.T, testcase string, oldStatus string, newStatus string) {
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an attendee in status " + oldStatus + " and a second user")
	loc, _ := tstRegisterAttendeeAndTransitionToStatus(t, testcase, oldStatus)
	token2 := tstValidUserToken(t, "101")

	docs.When("when the second user tries to change the first attendee's status to " + newStatus)
	body := status.StatusChangeDto{
		Status:  newStatus,
		Comment: testcase,
	}
	response := tstPerformPost(loc+"/status", tstRenderJson(body), token2)

	docs.Then("then the request is denied as unauthorized (403) and the appropriate error is returned")
	tstRequireErrorResponse(t, response, http.StatusForbidden, "auth.forbidden", "you are not allowed to make this status transition - the attempt has been logged")

	docs.Then("and the status is unchanged")
	tstVerifyStatus(t, loc, oldStatus)

	docs.Then("and no dues or payment changes have been recorded")
	require.Empty(t, paymentMock.Recording())

	docs.Then("and no email messages have been sent")
	require.Empty(t, mailMock.Recording())
}

func tstStatusChange_Regdesk_Deny(t *testing.T, testcase string, oldStatus string, newStatus string) {
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an attendee in status " + oldStatus + " and a second attendee with the regdesk permission")
	loc, _ := tstRegisterAttendeeAndTransitionToStatus(t, testcase, oldStatus)
	regdeskUserToken := tstRegisterRegdeskAttendee(t, testcase)

	docs.When("when the regdesk attendee tries to change the first attendee's status to " + newStatus)
	body := status.StatusChangeDto{
		Status:  newStatus,
		Comment: testcase,
	}
	response := tstPerformPost(loc+"/status", tstRenderJson(body), regdeskUserToken)

	docs.Then("then the request is denied as unauthorized (403) and the appropriate error is returned")
	tstRequireErrorResponse(t, response, http.StatusForbidden, "auth.forbidden", "you are not allowed to make this status transition - the attempt has been logged")

	docs.Then("and the status is unchanged")
	tstVerifyStatus(t, loc, oldStatus)

	docs.Then("and no dues or payment changes have been recorded")
	require.Empty(t, paymentMock.Recording())

	docs.Then("and no email messages have been sent")
	require.Empty(t, mailMock.Recording())
}

func tstStatusChange_Regdesk_Unavailable(t *testing.T, testcase string,
	oldStatus string, newStatus string,
	injectExtraTransactions []paymentservice.Transaction,
	message string, details string,
) {
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an attendee in status " + oldStatus + " and a second attendee with the regdesk permission")
	loc, _ := tstRegisterAttendeeAndTransitionToStatus(t, testcase, oldStatus)
	for _, tx := range injectExtraTransactions {
		_ = paymentMock.InjectTransaction(context.Background(), tx)
	}
	regdeskUserToken := tstRegisterRegdeskAttendee(t, testcase)

	docs.When("when the regdesk attendee prematurely tries to change the first attendee's status to " + newStatus)
	body := status.StatusChangeDto{
		Status:  newStatus,
		Comment: testcase,
	}
	response := tstPerformPost(loc+"/status", tstRenderJson(body), regdeskUserToken)

	docs.Then("then the request fails as conflict (409) and the appropriate error is returned")
	tstRequireErrorResponse(t, response, http.StatusConflict, message, details)

	docs.Then("and the status is unchanged")
	tstVerifyStatus(t, loc, oldStatus)

	docs.Then("and no dues or payment changes have been recorded")
	require.Empty(t, paymentMock.Recording())

	docs.Then("and no email messages have been sent")
	require.Empty(t, mailMock.Recording())
}

func tstStatusChange_Regdesk_Allow(t *testing.T, testcase string,
	oldStatus string, newStatus string,
	expectedTransactions []paymentservice.Transaction,
	expectedMailRequests []mailservice.TemplateRequestDto,
) {
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an attendee in status " + oldStatus + " and a second attendee with the regdesk permission")
	loc, _ := tstRegisterAttendeeAndTransitionToStatus(t, testcase, oldStatus)
	regdeskUserToken := tstRegisterRegdeskAttendee(t, testcase)

	docs.When("when the regdesk attendee changes the first attendee's status to " + newStatus)
	body := status.StatusChangeDto{
		Status:  newStatus,
		Comment: testcase,
	}
	response := tstPerformPost(loc+"/status", tstRenderJson(body), regdeskUserToken)

	docs.Then("then the request is successful and the status change has been done")
	require.Equal(t, http.StatusNoContent, response.status)
	tstVerifyStatus(t, loc, newStatus)

	docs.Then("and the appropriate dues were booked in the payment service")
	require.Equal(t, len(expectedTransactions), len(paymentMock.Recording()))
	for i, expected := range expectedTransactions {
		actual := paymentMock.Recording()[i]
		expected.DueDate = actual.DueDate // TODO remove when due date logic implemented
		require.EqualValues(t, expected, actual)
	}

	docs.Then("and the appropriate email messages were sent via the mail service")
	require.Equal(t, len(expectedMailRequests), len(mailMock.Recording()))
	for i, expected := range expectedMailRequests {
		actual := mailMock.Recording()[i]
		require.Contains(t, actual.Email, expected.Email)
		actual.Email = expected.Email
		require.EqualValues(t, expected, actual)
	}
}

func tstStatusChange_Staff_Other_Deny(t *testing.T, testcase string, oldStatus string, newStatus string) {
	tstSetup(tstConfigFile(false, true, true))
	defer tstShutdown()

	docs.Given("given an attendee in status " + oldStatus + " and a second user who is staff")
	loc, _ := tstRegisterAttendeeAndTransitionToStatus(t, testcase, oldStatus)
	token := tstValidStaffToken(t, "202")

	docs.When("when the staffer tries to change the first attendee's status to " + newStatus)
	body := status.StatusChangeDto{
		Status:  newStatus,
		Comment: testcase,
	}
	response := tstPerformPost(loc+"/status", tstRenderJson(body), token)

	docs.Then("then the request is denied as unauthorized (403) and the appropriate error is returned, because staff have no special privileges for status changes")
	tstRequireErrorResponse(t, response, http.StatusForbidden, "auth.forbidden", "you are not allowed to make this status transition - the attempt has been logged")

	docs.Then("and the status is unchanged")
	tstVerifyStatus(t, loc, oldStatus)

	docs.Then("and no dues or payment changes have been recorded")
	require.Empty(t, paymentMock.Recording())

	docs.Then("and no email messages have been sent")
	require.Empty(t, mailMock.Recording())
}

// admins never get deny (403), but they can get "not possible right now" (409)

func tstStatusChange_Admin_Unavailable(t *testing.T, testcase string,
	oldStatus string, newStatus string,
	injectExtraTransactions []paymentservice.Transaction,
	message string, details string) {
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an attendee in status " + oldStatus)
	loc, _ := tstRegisterAttendeeAndTransitionToStatus(t, testcase, oldStatus)
	for _, tx := range injectExtraTransactions {
		_ = paymentMock.InjectTransaction(context.Background(), tx)
	}

	docs.When("when an admin prematurely tries to change the status to " + newStatus)
	body := status.StatusChangeDto{
		Status:  newStatus,
		Comment: testcase,
	}
	response := tstPerformPost(loc+"/status", tstRenderJson(body), tstValidAdminToken(t))

	docs.Then("then the request fails as conflict (409) and the appropriate error is returned")
	tstRequireErrorResponse(t, response, http.StatusConflict, message, details)

	docs.Then("and the status is unchanged")
	tstVerifyStatus(t, loc, oldStatus)

	docs.Then("and no dues or payment changes have been recorded")
	require.Empty(t, paymentMock.Recording())

	docs.Then("and no email messages have been sent")
	require.Empty(t, mailMock.Recording())
}

func tstStatusChange_Admin_Allow(t *testing.T, testcase string,
	oldStatus string, newStatus string,
	injectExtraTransactions []paymentservice.Transaction,
	expectedTransactions []paymentservice.Transaction,
	expectedMailRequests []mailservice.TemplateRequestDto,
) {
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an attendee in status " + oldStatus)
	loc, _ := tstRegisterAttendeeAndTransitionToStatus(t, testcase, oldStatus)
	for _, tx := range injectExtraTransactions {
		_ = paymentMock.InjectTransaction(context.Background(), tx)
	}

	docs.When("when an admin changes their status to " + newStatus)
	body := status.StatusChangeDto{
		Status:  newStatus,
		Comment: testcase,
	}
	response := tstPerformPost(loc+"/status", tstRenderJson(body), tstValidAdminToken(t))

	docs.Then("then the request is successful and the status change has been done")
	require.Equal(t, http.StatusNoContent, response.status)
	tstVerifyStatus(t, loc, newStatus)

	docs.Then("and the appropriate dues were booked in the payment service")
	require.Equal(t, len(expectedTransactions), len(paymentMock.Recording()))
	for i, expected := range expectedTransactions {
		actual := paymentMock.Recording()[i]
		expected.DueDate = actual.DueDate // TODO remove when due date logic implemented
		require.EqualValues(t, expected, actual)
	}

	docs.Then("and the appropriate email messages were sent via the mail service")
	require.Equal(t, len(expectedMailRequests), len(mailMock.Recording()))
	for i, expected := range expectedMailRequests {
		actual := mailMock.Recording()[i]
		require.Contains(t, actual.Email, expected.Email)
		actual.Email = expected.Email
		require.EqualValues(t, expected, actual)
	}
}

// TODO test unbook unpaid dues on cancel (but not paid dues!), in order of invoicing (don't forget negative dues in history)

// TODO test downstream errors (502) by simulating errors in payment and mail service

// helper functions

func tstRequireAttendeeStatus(t *testing.T, expected string, responseBody string) {
	statusDto := status.StatusDto{}
	tstParseJson(responseBody, &statusDto)

	expectedStatusDto := status.StatusDto{
		Status: expected,
	}
	require.EqualValues(t, expectedStatusDto, statusDto, "status did not match expected value")
}

func tstRegisterRegdeskAttendee(t *testing.T, testcase string) string {
	token := tstValidUserToken(t, "101")

	loc2, _ := tstRegisterAttendeeWithToken(t, testcase+"second", token)
	permBody := admin.AdminInfoDto{
		Permissions: "regdesk",
	}
	permissionResponse := tstPerformPut(loc2+"/admin", tstRenderJson(permBody), tstValidAdminToken(t))
	require.Equal(t, http.StatusNoContent, permissionResponse.status)

	return token
}

func tstRegisterAttendeeAndTransitionToStatus(t *testing.T, testcase string, status string) (location string, att attendee.AttendeeDto) {
	// this works in all configurations, and for status changes, it makes no difference if a user is staff
	token := tstValidStaffToken(t, "1")

	location, att = tstRegisterAttendeeWithToken(t, testcase, token)
	if status == "new" {
		return
	}

	ctx := context.Background()
	attid, _ := strconv.Atoi(att.Id)

	// approved
	_ = database.GetRepository().AddStatusChange(ctx, tstCreateStatusChange(attid, "approved"))
	_ = paymentMock.InjectTransaction(ctx, tstCreateTransaction(attid, paymentservice.Due, 25500))
	if status == "approved" {
		return
	}

	if status == "deleted" {
		_ = database.GetRepository().AddStatusChange(ctx, tstCreateStatusChange(attid, "deleted"))
		_ = paymentMock.InjectTransaction(ctx, tstCreateTransaction(attid, paymentservice.Due, -25500))
		return
	}

	// partially paid
	_ = database.GetRepository().AddStatusChange(ctx, tstCreateStatusChange(attid, "partially paid"))
	_ = paymentMock.InjectTransaction(ctx, tstCreateTransaction(attid, paymentservice.Payment, 15500))
	if status == "partially paid" {
		return
	}

	// paid
	_ = database.GetRepository().AddStatusChange(ctx, tstCreateStatusChange(attid, "paid"))
	_ = paymentMock.InjectTransaction(ctx, tstCreateTransaction(attid, paymentservice.Payment, 10000))
	if status == "paid" {
		return
	}

	// checked in
	_ = database.GetRepository().AddStatusChange(ctx, tstCreateStatusChange(attid, "checked in"))
	if status == "checked in" {
		return
	}

	// cancelled
	_ = database.GetRepository().AddStatusChange(ctx, tstCreateStatusChange(attid, "cancelled"))
	if status == "cancelled" {
		return
	}

	// invalid status - error in test code
	t.FailNow()
	return
}

func tstCreateStatusChange(attid int, status string) *entity.StatusChange {
	return &entity.StatusChange{
		AttendeeId: uint(attid),
		Status:     status,
	}
}

func tstCreateTransaction(attid int, ty paymentservice.TransactionType, amount int64) paymentservice.Transaction {
	method := paymentservice.Internal
	if ty == paymentservice.Payment {
		method = paymentservice.Credit
	}
	return paymentservice.Transaction{
		ID:        "1234-1234abc",
		DebitorID: uint(attid),
		Type:      ty,
		Method:    method,
		Amount: paymentservice.Amount{
			Currency:  "EUR",
			GrossCent: amount,
			VatRate:   19,
		},
		Status:        paymentservice.Valid,
		EffectiveDate: "1999-12-31",
		DueDate:       time.Now(),
		Deletion:      nil,
	}
}

func tstCreateMatcherTransaction(attid int, ty paymentservice.TransactionType, amount int64, comment string) paymentservice.Transaction {
	method := paymentservice.Internal
	if ty == paymentservice.Payment {
		method = paymentservice.Credit
	}
	return paymentservice.Transaction{
		ID:        "",
		DebitorID: uint(attid),
		Type:      ty,
		Method:    method,
		Amount: paymentservice.Amount{
			Currency:  "EUR",
			GrossCent: amount,
			VatRate:   19,
		},
		Status:        paymentservice.Valid,
		EffectiveDate: "",         // TODO
		DueDate:       time.Now(), // TODO
		Deletion:      nil,        // TODO
		Comment:       comment,
	}
}

func tstVerifyStatus(t *testing.T, loc string, expectedStatus string) {
	response := tstPerformGet(loc+"/status", tstValidAdminToken(t))
	require.Equal(t, http.StatusOK, response.status)
	statusDto := status.StatusDto{}
	tstParseJson(response.body, &statusDto)
	require.Equal(t, expectedStatus, statusDto.Status)
}
