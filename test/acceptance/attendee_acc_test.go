package acceptance

import (
	"github.com/eurofurence/reg-attendee-service/internal/api/v1/attendee"
	"net/http"
	"net/url"
	"testing"

	"github.com/eurofurence/reg-attendee-service/docs"
	"github.com/stretchr/testify/require"
)

// ------------------------------------------
// acceptance tests for the attendee resource
// ------------------------------------------

// --- create new attendee ---

// -- validation and duplicate handling tests --

func TestCreateNewAttendeeInvalid(t *testing.T) {
	docs.Given("given the configuration for public standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an unauthenticated user")

	docs.When("when they create a new attendee with invalid data")
	attendeeSent := tstBuildValidAttendee("nav1-")
	attendeeSent.Nickname = "$%&^@!$"
	attendeeSent.Packages = attendeeSent.Packages + ",sponsor" // a constraint violation
	attendeeSent.Birthday = "2004-11-23"                       // too young
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), tstNoToken())

	docs.Then("then the attendee is rejected with an appropriate error response")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "attendee.data.invalid", url.Values{
		"birthday": []string{"birthday must be no earlier than 1901-01-01 and no later than 2001-08-14"},
		"nickname": []string{"nickname field must contain at least one alphanumeric character", "nickname field must not contain more than two non-alphanumeric characters (not counting spaces)"},
		"packages": []string{"cannot pick both sponsor2 and sponsor - constraint violated"},
	})
}

func TestCreateNewAttendeeSyntaxInvalid(t *testing.T) {
	docs.Given("given the configuration for public standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an unauthenticated user")

	docs.When("when they try to create a new attendee with syntactically invalid data")
	attendeeSent := tstBuildValidAttendee("nav2-")
	syntaxErrorJson := "{" + tstRenderJson(attendeeSent)
	response := tstPerformPost("/api/rest/v1/attendees", syntaxErrorJson, tstNoToken())

	docs.Then("then the attendee is rejected with an appropriate error response")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "attendee.parse.error", url.Values{})
}

func TestCreateNewAttendeeAdminOnlyFlag(t *testing.T) {
	docs.Given("given the configuration for standard public registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an unauthenticated user")

	docs.When("when they send a new attendee and attempt to set an admin only flag (guest)")
	attendeeSent := tstBuildValidAttendee("nav3-")
	attendeeSent.Flags = "guest,hc"
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), tstNoToken())

	docs.Then("then the attendee is rejected with an error response")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "attendee.data.invalid", url.Values{
		"flags": []string{"flags field must be a comma separated combination of any of anon,ev,hc"},
	})
}

func TestCreateNewAttendeeReadOnlyFlag(t *testing.T) {
	docs.Given("given the configuration for standard public registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an unauthenticated user")

	docs.When("when they send a new attendee and attempt to set a read only flag (ev)")
	attendeeSent := tstBuildValidAttendee("nav4-")
	attendeeSent.Flags = "ev,anon"
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), tstNoToken())

	docs.Then("then the attendee is rejected with an error response")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "attendee.data.invalid", url.Values{
		"flags": []string{"forbidden change in state of choice key ev - only an admin can do that"},
	})
}

func TestCreateNewAttendeeAdminOnlyFlag_Admin(t *testing.T) {
	docs.Given("given the configuration for standard public registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an admin")
	token := tstValidAdminToken(t)

	docs.When("when they send a new attendee and attempt to set an admin only flag (guest)")
	attendeeSent := tstBuildValidAttendee("nav5-")
	attendeeSent.Flags = "guest"
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), token)

	docs.Then("then the attendee is rejected with an error response, because admin only flags belong in adminInfo")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "attendee.data.invalid", url.Values{
		"flags": []string{"flags field must be a comma separated combination of any of anon,ev,hc"},
	})
}

func TestCreateNewAttendeeDefaultReadOnlyPackage(t *testing.T) {
	docs.Given("given the configuration for staff pre-registration before public registration")
	tstSetup(tstConfigFile(false, true, true))
	defer tstShutdown()

	docs.Given("given a staffer")
	staffToken := tstValidStaffToken(t, "1")

	docs.When("when they send a new attendee and attempt to leave out a read only default package (room-none)")
	attendeeSent := tstBuildValidAttendee("nav6-")
	attendeeSent.Packages = "attendance,stage,sponsor"
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), staffToken)

	docs.Then("then the attendee is rejected with an error response")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "attendee.data.invalid", url.Values{
		"packages": []string{"forbidden change in state of choice key room-none - only an admin can do that"},
	})
}

func TestCreateNewAttendeeDuplicateHandling(t *testing.T) {
	docs.Given("given the configuration for standard public registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an unauthenticated user and an existing registration")
	attendeeSent := tstBuildValidAttendee("na10-")
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), tstNoToken())
	require.Equal(t, http.StatusCreated, response.status, "unexpected http response status")
	require.Regexp(t, "^\\/api\\/rest\\/v1\\/attendees\\/[1-9][0-9]*$", response.location, "invalid location header in response")

	docs.When("when they attempt to create a new attendee with the same nickname, zip, email")
	duplicateAttendeeSent := tstBuildValidAttendee("na10-")
	duplicateAttendeeSent.Nickname = attendeeSent.Nickname
	duplicateAttendeeSent.Zip = attendeeSent.Zip
	duplicateAttendeeSent.Email = attendeeSent.Email
	duplicateResponse := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(duplicateAttendeeSent), tstNoToken())

	docs.Then("then the attendee is rejected with an error response indicating a duplicate")
	tstRequireErrorResponse(t, duplicateResponse, http.StatusConflict, "attendee.data.duplicate", url.Values{
		"attendee": []string{"there is already an attendee with this information (looking at nickname, email, and zip code)"},
	})
}

// -- no login required for new registrations --

// - before both target times -

func TestCreateNewAttendee_NoLoginRequired_TooEarly_Anon(t *testing.T) {
	docs.Given("given the configuration for public registration before any registration target time")
	tstSetup(tstConfigFile(false, false, false))
	defer tstShutdown()

	docs.Given("given an unauthenticated user")

	docs.When("when they attempt to create a new attendee with valid data before registration has begun")
	attendeeSent := tstBuildValidAttendee("na1-")
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), tstNoToken())

	docs.Then("then the attempt is rejected with an appropriate error response")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "attendee.data.invalid", url.Values{
		"timing": []string{"public registration has not opened at this time, please come back later"},
	})
}

func TestCreateNewAttendee_NoLoginRequired_TooEarly_User(t *testing.T) {
	docs.Given("given the configuration for public registration before any registration target time")
	tstSetup(tstConfigFile(false, false, false))
	defer tstShutdown()

	docs.Given("given a logged in user")
	token := tstValidUserToken(t, "101")

	docs.When("when they attempt to create a new attendee with valid data before registration has begun")
	attendeeSent := tstBuildValidAttendee("na2-")
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), token)

	docs.Then("then the attempt is rejected with an appropriate error response")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "attendee.data.invalid", url.Values{
		"timing": []string{"public registration has not opened at this time, please come back later"},
	})
}

func TestCreateNewAttendee_NoLoginRequired_TooEarly_Staff(t *testing.T) {
	docs.Given("given the configuration for staff registration before any registration target time")
	tstSetup(tstConfigFile(false, true, false))
	defer tstShutdown()

	docs.Given("given a logged in user who is staff")
	token := tstValidStaffToken(t, "202")

	docs.When("when they attempt to create a new attendee with valid data before even staff registration has begun")
	attendeeSent := tstBuildValidAttendee("na3-")
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), token)

	docs.Then("then the attempt is rejected with an appropriate error response")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "attendee.data.invalid", url.Values{
		"timing": []string{"staff registration has not opened at this time, please come back later"},
	})
}

func TestCreateNewAttendee_NoLoginRequired_TooEarly_AdminIsLikeStaff(t *testing.T) {
	docs.Given("given the configuration for public registration before any registration target time")
	tstSetup(tstConfigFile(false, false, false))
	defer tstShutdown()

	docs.Given("given a logged in admin")
	token := tstValidAdminToken(t)

	docs.When("when they attempt to create a new attendee with valid data before registration has begun")
	attendeeSent := tstBuildValidAttendee("na4-")
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), token)

	docs.Then("then the attempt is rejected with an appropriate error response, that is, admins are treated just like staff")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "attendee.data.invalid", url.Values{
		"timing": []string{"public registration has not opened at this time, please come back later"},
	})
}

// - between both target times -

func TestCreateNewAttendee_NoLoginRequired_Between_Anon(t *testing.T) {
	docs.Given("given the configuration for staff pre-registration before public registration but after the staff start time")
	tstSetup(tstConfigFile(false, true, true))
	defer tstShutdown()

	docs.Given("given an unauthenticated user")

	docs.When("when they attempt to create a new attendee with valid data")
	attendeeSent := tstBuildValidAttendee("na10-")
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), tstNoToken())

	docs.Then("then the request is denied with the appropriate error response and no location header is supplied")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "attendee.data.invalid", url.Values{
		"timing": []string{"public registration has not opened at this time, please come back later"},
	})
	require.Equal(t, "", response.location, "non-empty location header in response")
}

func TestCreateNewAttendee_NoLoginRequired_Between_User(t *testing.T) {
	docs.Given("given the configuration for staff pre-registration before public registration but after the staff start time")
	tstSetup(tstConfigFile(false, true, true))
	defer tstShutdown()

	docs.Given("given an authenticated regular user")
	userToken := tstValidUserToken(t, "1")

	docs.When("when they attempt to create a new attendee with valid data")
	attendeeSent := tstBuildValidAttendee("na11-")
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), userToken)

	docs.Then("then the request is denied with the appropriate error response and no location header is supplied")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "attendee.data.invalid", url.Values{
		"timing": []string{"public registration has not opened at this time, please come back later"},
	})
	require.Equal(t, "", response.location, "non-empty location header in response")
}

func TestCreateNewAttendee_NoLoginRequired_Between_Staff(t *testing.T) {
	docs.Given("given the configuration for staff pre-registration before public registration but after the staff start time")
	tstSetup(tstConfigFile(false, true, true))
	defer tstShutdown()

	docs.Given("given a staffer")
	staffToken := tstValidStaffToken(t, "1")

	docs.When("when they attempt to create a new attendee with valid data")
	attendeeSent := tstBuildValidAttendee("na12-")
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), staffToken)

	docs.Then("then the attendee is successfully created")
	require.Equal(t, http.StatusCreated, response.status, "unexpected http response status")
	require.Regexp(t, "^\\/api\\/rest\\/v1\\/attendees\\/[1-9][0-9]*$", response.location, "invalid location header in response")
}

func TestCreateNewAttendee_NoLoginRequired_Between_AdminIsLikeStaff(t *testing.T) {
	docs.Given("given the configuration for staff pre-registration before public registration but after the staff start time")
	tstSetup(tstConfigFile(false, true, true))
	defer tstShutdown()

	docs.Given("given an admin")
	adminToken := tstValidAdminToken(t)

	docs.When("when they attempt to create a new attendee with valid data")
	attendeeSent := tstBuildValidAttendee("na13-")
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), adminToken)

	docs.Then("then the attendee is successfully created, that is an admin is treated just like staff")
	require.Equal(t, http.StatusCreated, response.status, "unexpected http response status")
	require.Regexp(t, "^\\/api\\/rest\\/v1\\/attendees\\/[1-9][0-9]*$", response.location, "invalid location header in response")
}

// - after public target time -

func TestCreateNewAttendee_NoLoginRequired_After_Anon(t *testing.T) {
	docs.Given("given the configuration for public standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an unauthenticated user")

	docs.When("when they create a new attendee")
	attendeeSent := tstBuildValidAttendee("na20-")
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), tstNoToken())

	docs.Then("then the attendee is successfully created")
	require.Equal(t, http.StatusCreated, response.status, "unexpected http response status")
	require.Regexp(t, "^\\/api\\/rest\\/v1\\/attendees\\/[1-9][0-9]*$", response.location, "invalid location header in response")

	docs.Then("and its data can be read again by an admin")
	attendeeReadAgain := tstReadAttendee(t, response.location)
	// difference in id is ok, so copy it over to expected
	attendeeSent.Id = attendeeReadAgain.Id
	require.EqualValues(t, attendeeSent, attendeeReadAgain, "attendee data read did not match sent data")
}

func TestCreateNewAttendee_NoLoginRequired_After_User(t *testing.T) {
	docs.Given("given the configuration for public registration after normal reg is open")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given a logged in user")
	token := tstValidUserToken(t, "101")

	docs.When("when they attempt to create a new attendee with valid data after public registration has begun")
	attendeeSent := tstBuildValidAttendee("na21-")
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), token)

	docs.Then("then the attendee is successfully created")
	require.Equal(t, http.StatusCreated, response.status, "unexpected http response status")
	require.Regexp(t, "^\\/api\\/rest\\/v1\\/attendees\\/[1-9][0-9]*$", response.location, "invalid location header in response")

	docs.Then("and its data can be read again by the same user")
	readAgainResponse := tstPerformGet(response.location, token)
	attendeeReadAgain := attendee.AttendeeDto{}
	tstParseJson(readAgainResponse.body, &attendeeReadAgain)
	// difference in id is ok, so copy it over to expected
	attendeeSent.Id = attendeeReadAgain.Id
	require.EqualValues(t, attendeeSent, attendeeReadAgain, "attendee data read did not match sent data")
}

func TestCreateNewAttendee_NoLoginRequired_After_Staff(t *testing.T) {
	docs.Given("given the configuration for public registration after normal reg is open")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given a logged in staffer")
	token := tstValidStaffToken(t, "202")

	docs.When("when they attempt to create a new attendee with valid data after public registration has begun")
	attendeeSent := tstBuildValidAttendee("na22-")
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), token)

	docs.Then("then the attendee is successfully created")
	require.Equal(t, http.StatusCreated, response.status, "unexpected http response status")
	require.Regexp(t, "^\\/api\\/rest\\/v1\\/attendees\\/[1-9][0-9]*$", response.location, "invalid location header in response")
}

func TestCreateNewAttendee_NoLoginRequired_After_AdminIsLikeStaff(t *testing.T) {
	docs.Given("given the configuration for public registration after normal reg is open")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given a logged in admin")
	token := tstValidAdminToken(t)

	docs.When("when they attempt to create a new attendee with valid data after public registration has begun")
	attendeeSent := tstBuildValidAttendee("na23-")
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), token)

	docs.Then("then the attendee is successfully created, that is, admins are treated just like staff")
	require.Equal(t, http.StatusCreated, response.status, "unexpected http response status")
	require.Regexp(t, "^\\/api\\/rest\\/v1\\/attendees\\/[1-9][0-9]*$", response.location, "invalid location header in response")
}

// -- login required for all new registrations --

// we only test the timing-related cases, validation is no different

// - before both target times -

func TestCreateNewAttendee_LoginRequired_TooEarly_Anon(t *testing.T) {
	docs.Given("given the configuration for login-only registration before any registration target time")
	tstSetup(tstConfigFile(true, false, false))
	defer tstShutdown()

	docs.Given("given an unauthenticated user")

	docs.When("when they attempt to create a new attendee with valid data before public registration has begun")
	attendeeSent := tstBuildValidAttendee("na30-")
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), tstNoToken())

	docs.Then("then the attempt is rejected as unauthenticated (401) with an appropriate error response")
	tstRequireErrorResponse(t, response, http.StatusUnauthorized, "auth.unauthorized", "you must be logged in for this operation")
}

func TestCreateNewAttendee_LoginRequired_TooEarly_User(t *testing.T) {
	docs.Given("given the configuration for login-only registration before any registration target time")
	tstSetup(tstConfigFile(true, false, false))
	defer tstShutdown()

	docs.Given("given a logged in user")
	token := tstValidUserToken(t, "1")

	docs.When("when they attempt to create a new attendee with valid data before public registration has begun")
	attendeeSent := tstBuildValidAttendee("na31-")
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), token)

	docs.Then("then the attempt is rejected as too early with an appropriate error response")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "attendee.data.invalid", url.Values{
		"timing": []string{"public registration has not opened at this time, please come back later"},
	})
}

func TestCreateNewAttendee_LoginRequired_TooEarly_Staff(t *testing.T) {
	docs.Given("given the configuration for login-only registration before any registration target time")
	tstSetup(tstConfigFile(true, true, false))
	defer tstShutdown()

	docs.Given("given a logged in staffer")
	token := tstValidStaffToken(t, "1")

	docs.When("when they attempt to create a new attendee with valid data before public registration has begun")
	attendeeSent := tstBuildValidAttendee("na32-")
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), token)

	docs.Then("then the attempt is rejected with an appropriate error response")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "attendee.data.invalid", url.Values{
		"timing": []string{"staff registration has not opened at this time, please come back later"},
	})
}

func TestCreateNewAttendee_LoginRequired_TooEarly_AdminIsLikeStaff(t *testing.T) {
	docs.Given("given the configuration for login-only registration before any registration target time")
	tstSetup(tstConfigFile(true, true, false))
	defer tstShutdown()

	docs.Given("given a logged in admin")
	token := tstValidAdminToken(t)

	docs.When("when they attempt to create a new attendee with valid data before public registration has begun")
	attendeeSent := tstBuildValidAttendee("na33-")
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), token)

	docs.Then("then the attempt is rejected with an appropriate error response, that is, admins are treated just like staff")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "attendee.data.invalid", url.Values{
		"timing": []string{"staff registration has not opened at this time, please come back later"},
	})
}

// - between both target times -

func TestCreateNewAttendee_LoginRequired_Between_Anon(t *testing.T) {
	docs.Given("given the configuration for login-only registration after staff reg is open but before normal reg is open")
	tstSetup(tstConfigFile(true, true, true))
	defer tstShutdown()

	docs.Given("given an unauthenticated user")

	docs.When("when they attempt to create a new attendee with valid data before public registration has begun")
	attendeeSent := tstBuildValidAttendee("na40-")
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), tstNoToken())

	docs.Then("then the attempt is rejected as unauthenticated (401) with an appropriate error response")
	tstRequireErrorResponse(t, response, http.StatusUnauthorized, "auth.unauthorized", "you must be logged in for this operation")
}

func TestCreateNewAttendee_LoginRequired_Between_User(t *testing.T) {
	docs.Given("given the configuration for login-only registration after staff reg is open but before normal reg is open")
	tstSetup(tstConfigFile(true, true, true))
	defer tstShutdown()

	docs.Given("given a logged in user")
	token := tstValidUserToken(t, "1")

	docs.When("when they attempt to create a new attendee with valid data before public registration has begun")
	attendeeSent := tstBuildValidAttendee("na41-")
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), token)

	docs.Then("then the attempt is rejected as too early with an appropriate error response")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "attendee.data.invalid", url.Values{
		"timing": []string{"public registration has not opened at this time, please come back later"},
	})
}

func TestCreateNewAttendee_LoginRequired_Between_Staff(t *testing.T) {
	docs.Given("given the configuration for login-only registration after staff reg is open but before normal reg is open")
	tstSetup(tstConfigFile(true, true, true))
	defer tstShutdown()

	docs.Given("given a logged in staffer")
	token := tstValidStaffToken(t, "1")

	docs.When("when they attempt to create a new attendee with valid data after staff registration has begun")
	attendeeSent := tstBuildValidAttendee("na42-")
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), token)

	docs.Then("then the attendee is successfully created")
	require.Equal(t, http.StatusCreated, response.status, "unexpected http response status")
	require.Regexp(t, "^\\/api\\/rest\\/v1\\/attendees\\/[1-9][0-9]*$", response.location, "invalid location header in response")
}

func TestCreateNewAttendee_LoginRequired_Between_AdminIsLikeStaff(t *testing.T) {
	docs.Given("given the configuration for login-only registration after staff reg is open but before normal reg is open")
	tstSetup(tstConfigFile(true, true, true))
	defer tstShutdown()

	docs.Given("given a logged in admin")
	token := tstValidAdminToken(t)

	docs.When("when they attempt to create a new attendee with valid data after staff registration has begun")
	attendeeSent := tstBuildValidAttendee("na43-")
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), token)

	docs.Then("then the attendee is successfully created, that is, admins are treated just like staff")
	require.Equal(t, http.StatusCreated, response.status, "unexpected http response status")
	require.Regexp(t, "^\\/api\\/rest\\/v1\\/attendees\\/[1-9][0-9]*$", response.location, "invalid location header in response")
}

// - after public target time -

func TestCreateNewAttendee_LoginRequired_After_Anon(t *testing.T) {
	docs.Given("given the configuration for login-only registration after normal reg is open")
	tstSetup(tstConfigFile(true, false, true))
	defer tstShutdown()

	docs.Given("given an unauthenticated user")

	docs.When("when they attempt to create a new attendee with valid data after public registration has begun")
	attendeeSent := tstBuildValidAttendee("na50-")
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), tstNoToken())

	docs.Then("then the attempt is rejected as unauthenticated (401) with an appropriate error response")
	tstRequireErrorResponse(t, response, http.StatusUnauthorized, "auth.unauthorized", "you must be logged in for this operation")
}

func TestCreateNewAttendee_LoginRequired_After_User(t *testing.T) {
	docs.Given("given the configuration for login-only registration after normal reg is open")
	tstSetup(tstConfigFile(true, false, true))
	defer tstShutdown()

	docs.Given("given a logged in user")
	token := tstValidUserToken(t, "1")

	docs.When("when they attempt to create a new attendee with valid data after public registration has begun")
	attendeeSent := tstBuildValidAttendee("na51-")
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), token)

	docs.Then("then the attendee is successfully created")
	require.Equal(t, http.StatusCreated, response.status, "unexpected http response status")
	require.Regexp(t, "^\\/api\\/rest\\/v1\\/attendees\\/[1-9][0-9]*$", response.location, "invalid location header in response")
}

func TestCreateNewAttendee_LoginRequired_After_Staff(t *testing.T) {
	docs.Given("given the configuration for login-only registration after normal reg is open")
	tstSetup(tstConfigFile(true, false, true))
	defer tstShutdown()

	docs.Given("given a logged in staffer")
	token := tstValidStaffToken(t, "1")

	docs.When("when they attempt to create a new attendee with valid data after public registration has begun")
	attendeeSent := tstBuildValidAttendee("na52-")
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), token)

	docs.Then("then the attendee is successfully created")
	require.Equal(t, http.StatusCreated, response.status, "unexpected http response status")
	require.Regexp(t, "^\\/api\\/rest\\/v1\\/attendees\\/[1-9][0-9]*$", response.location, "invalid location header in response")
}

func TestCreateNewAttendee_LoginRequired_After_AdminIsLikeStaff(t *testing.T) {
	docs.Given("given the configuration for login-only registration after normal reg is open")
	tstSetup(tstConfigFile(true, false, true))
	defer tstShutdown()

	docs.Given("given a logged in admin")
	token := tstValidAdminToken(t)

	docs.When("when they attempt to create a new attendee with valid data after public registration has begun")
	attendeeSent := tstBuildValidAttendee("na53-")
	response := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(attendeeSent), token)

	docs.Then("then the attendee is successfully created, that is, admins are treated just like staff")
	require.Equal(t, http.StatusCreated, response.status, "unexpected http response status")
	require.Regexp(t, "^\\/api\\/rest\\/v1\\/attendees\\/[1-9][0-9]*$", response.location, "invalid location header in response")
}

// --- update attendee ---

func TestUpdateExistingAttendee_Self(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an existing attendee")
	token := tstValidUserToken(t, "101")
	location1, attendee1 := tstRegisterAttendeeWithToken(t, "ua1-", token)

	docs.When("when they send updated attendee info while logged in")
	changedAttendee := attendee1
	changedAttendee.FirstName = "Eva"
	changedAttendee.LastName = "Musterfrau"
	updateResponse := tstPerformPut(location1, tstRenderJson(changedAttendee), token)

	docs.Then("then the attendee is successfully updated and the changed data can be read again")
	require.Equal(t, http.StatusOK, updateResponse.status, "unexpected http response status for update")
	require.Equal(t, location1, updateResponse.location, "location unexpectedly changed during update")
	attendeeReadAgain := tstReadAttendee(t, location1)
	require.EqualValues(t, changedAttendee, attendeeReadAgain, "attendee data read did not match updated data")
}

func TestUpdateExistingAttendee_Other(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given two existing users, the second of which has registered")
	location2, attendee2 := tstRegisterAttendee(t, "ua2b-")
	token := tstValidUserToken(t, "101")

	docs.When("when the first user sends updated attendee info for the second user, i.e. for someone else")
	changedAttendee := attendee2
	changedAttendee.FirstName = "Eva"
	changedAttendee.LastName = "Musterfrau"
	response := tstPerformPut(location2, tstRenderJson(changedAttendee), token)

	docs.Then("then the request is denied as unauthorized (403) and the data remains unchanged")
	tstRequireErrorResponse(t, response, http.StatusForbidden, "auth.forbidden", "you are not authorized to access this data - the attempt has been logged")
}

func TestUpdateExistingAttendeeSyntaxInvalid(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an existing attendee")
	location1, attendee1 := tstRegisterAttendee(t, "ua3-")

	docs.Given("given an admin")
	token := tstValidAdminToken(t)

	docs.When("when they try to update with syntactically invalid data")
	syntaxErrorJson := "{" + tstRenderJson(attendee1)
	response := tstPerformPut(location1, syntaxErrorJson, token)

	docs.Then("then the update is rejected with an appropriate error response")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "attendee.parse.error", "")
}

func TestUpdateExistingAttendeeDataInvalid(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an existing attendee who is logged in")
	location1, attendee1 := tstRegisterAttendee(t, "ua4-")

	docs.Given("given an admin")
	token := tstValidAdminToken(t)

	docs.When("when they try to update the information with invalid data")
	changedAttendee := attendee1
	changedAttendee.Nickname = "$%&^@!$"                             // not allowed
	changedAttendee.Packages = changedAttendee.Packages + ",sponsor" // a constraint violation
	changedAttendee.Birthday = "2004-11-23"                          // too young
	response := tstPerformPut(location1, tstRenderJson(changedAttendee), token)

	docs.Then("then the update is rejected with an appropriate error response")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "attendee.data.invalid", url.Values{
		"nickname": []string{"nickname field must contain at least one alphanumeric character", "nickname field must not contain more than two non-alphanumeric characters (not counting spaces)"},
		"packages": []string{"cannot pick both sponsor2 and sponsor - constraint violated"},
		"birthday": []string{"birthday must be no earlier than 1901-01-01 and no later than 2001-08-14"},
	})
}

func TestUpdateNonExistingAttendee(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an admin")
	token := tstValidAdminToken(t)

	docs.When("when they try to update a non-existent attendee")
	nonExistingAttendee := tstBuildValidAttendee("ua5-")
	response := tstPerformPut("/api/rest/v1/attendees/42", tstRenderJson(nonExistingAttendee), token)

	docs.Then("then the update is rejected with an appropriate error response")
	tstRequireErrorResponse(t, response, http.StatusNotFound, "attendee.id.notfound", "")
}

func TestUpdateAttendeeInvalidId(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an admin")
	token := tstValidAdminToken(t)

	docs.When("when they try to update an attendee with an invalid id")
	nonExistingAttendee := tstBuildValidAttendee("ua6-")
	response := tstPerformPut("/api/rest/v1/attendees/helloworld", tstRenderJson(nonExistingAttendee), token)

	docs.Then("then the update is rejected with an appropriate error response")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "attendee.id.invalid", "")
}

func TestDenyUpdateExistingAttendeeWhileNotLoggedIn(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an existing attendee and a user who is not logged in")
	existingAttendee := tstBuildValidAttendee("ua7-")
	existingAttendee.FirstName = "Marianne"
	creationResponse := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(existingAttendee), tstNoToken())
	require.Equal(t, http.StatusCreated, creationResponse.status, "unexpected http response status for create")
	attendeeReadAfterCreation := tstReadAttendee(t, creationResponse.location)

	docs.When("when they send updated attendee info while not logged in")
	changedAttendee := attendeeReadAfterCreation
	changedAttendee.FirstName = "Eva"
	updateResponse := tstPerformPut(creationResponse.location, tstRenderJson(changedAttendee), tstNoToken())

	docs.Then("then the request is denied as unauthenticated (401) and the data remains unchanged")
	require.Equal(t, http.StatusUnauthorized, updateResponse.status, "unexpected http response status for insecure update")
	attendeeReadAgain := tstReadAttendee(t, creationResponse.location)
	require.EqualValues(t, "Marianne", attendeeReadAgain.FirstName, "attendee data read did not match original data")
}

func TestDenyUpdateExistingOtherAttendeeWithStaffToken(t *testing.T) {
	docs.Given("given the configuration for staff pre-registration (the other config doesn't even have a staff token)")
	tstSetup(tstConfigFile(false, true, true))
	defer tstShutdown()

	docs.Given("given an existing attendee")
	existingAttendee := tstBuildValidAttendee("ua8-")
	existingAttendee.FirstName = "Marianne"
	creationResponse := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(existingAttendee), tstValidAdminToken(t))
	require.Equal(t, http.StatusCreated, creationResponse.status, "unexpected http response status for create")
	attendeeReadAfterCreation := tstReadAttendee(t, creationResponse.location)

	docs.When("when a logged in staffer, who is not that attendee, sends updated attendee info for them, i.e. for someone else")
	changedAttendee := attendeeReadAfterCreation
	changedAttendee.FirstName = "Eva"
	token := tstValidStaffToken(t, "202")
	updateResponse := tstPerformPut(creationResponse.location, tstRenderJson(changedAttendee), token)

	docs.Then("then the request is denied as unauthorized (403) and the data remains unchanged")
	require.Equal(t, http.StatusForbidden, updateResponse.status, "unexpected http response status for insecure update")
	attendeeReadAgain := tstReadAttendee(t, creationResponse.location)
	require.EqualValues(t, "Marianne", attendeeReadAgain.FirstName, "attendee data read did not match original data")
}

func TestUpdateExistingAttendeeAdminOnlyFlag(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an existing attendee who is logged in")
	location1, attendee1 := tstRegisterAttendee(t, "ua9-")
	token := tstValidUserToken(t, attendee1.Id)

	docs.When("when they send updated attendee info and attempt to add an admin-only flag (guest)")
	changedAttendee := attendee1
	changedAttendee.Flags = changedAttendee.Flags + ",guest"
	updateResponse := tstPerformPut(location1, tstRenderJson(changedAttendee), token)

	docs.Then("then the request is denied with an appropriate error")
	tstRequireErrorResponse(t, updateResponse, http.StatusBadRequest, "attendee.data.invalid", url.Values{
		"flags": []string{"flags field must be a comma separated combination of any of anon,ev,hc"},
	})

	docs.Then("and the data remains unchanged")
	attendeeReadAgain := tstReadAttendee(t, location1)
	require.EqualValues(t, "anon,hc", attendeeReadAgain.Flags, "attendee data read did not match original data")
}

func TestUpdateExistingAttendeeAdminOnlyFlag_Admin(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an existing attendee")
	location1, attendee1 := tstRegisterAttendee(t, "ua10-")

	docs.Given("given an admin")
	token := tstValidAdminToken(t)

	docs.When("when they send updated attendee info and attempt to add an admin-only flag (guest)")
	changedAttendee := attendee1
	changedAttendee.Flags = changedAttendee.Flags + ",guest"
	updateResponse := tstPerformPut(location1, tstRenderJson(changedAttendee), token)

	docs.Then("then the request is denied with an appropriate error, because admin only flags belong in adminInfo")
	tstRequireErrorResponse(t, updateResponse, http.StatusBadRequest, "attendee.data.invalid", url.Values{
		"flags": []string{"flags field must be a comma separated combination of any of anon,ev,hc"},
	})

	docs.Then("and the data remains unchanged")
	attendeeReadAgain := tstReadAttendee(t, location1)
	require.EqualValues(t, "anon,hc", attendeeReadAgain.Flags, "attendee data read did not match original data")
}

func TestUpdateExistingAttendeeReadOnlyFlag(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an existing attendee who is logged in")
	token := tstValidUserToken(t, "101")
	location1, attendee1 := tstRegisterAttendeeWithToken(t, "ua11-", token)

	docs.When("when they send updated attendee info and attempt to add a read-only flag (ev)")
	changedAttendee := attendee1
	changedAttendee.Flags = changedAttendee.Flags + ",ev"
	updateResponse := tstPerformPut(location1, tstRenderJson(changedAttendee), token)

	docs.Then("then the request is denied with an appropriate error, because only admins can change read only flags")
	tstRequireErrorResponse(t, updateResponse, http.StatusBadRequest, "attendee.data.invalid", url.Values{
		"flags": []string{"forbidden change in state of choice key ev - only an admin can do that"},
	})

	docs.Then("and the data remains unchanged")
	attendeeReadAgain := tstReadAttendee(t, location1)
	require.EqualValues(t, "anon,hc", attendeeReadAgain.Flags, "attendee data read did not match original data")
}

func TestUpdateExistingAttendeeReadOnlyFlag_Admin(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an existing attendee")
	location1, attendee1 := tstRegisterAttendee(t, "ua12-")

	docs.Given("given an admin")
	token := tstValidAdminToken(t)

	docs.When("when they send updated attendee info and add a read-only flag (ev)")
	changedAttendee := attendee1
	changedAttendee.Flags = changedAttendee.Flags + ",ev"
	updateResponse := tstPerformPut(location1, tstRenderJson(changedAttendee), token)

	docs.Then("then the attendee is successfully updated and the changed data can be read again")
	require.Equal(t, http.StatusOK, updateResponse.status, "unexpected http response status for update")
	require.Equal(t, location1, updateResponse.location, "location unexpectedly changed during update")
	attendeeReadAgain := tstReadAttendee(t, location1)
	require.EqualValues(t, changedAttendee, attendeeReadAgain, "attendee data read did not match updated data")
	require.EqualValues(t, "anon,hc,ev", attendeeReadAgain.Flags, "attendee data read did not match expected flags value")
}

// TODO test dues changes caused by attendee package updates and corresponding status changes

// --- get attendee ---

func TestDenyReadExistingAttendeeWhileNotLoggedIn(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an existing attendee")
	location1, _ := tstRegisterAttendee(t, "ga1-")

	docs.Given("given a user who is not logged in")

	docs.When("when they attempt to read attendee info while not logged in")
	readResponse := tstPerformGet(location1, tstNoToken())

	docs.Then("then the request is denied")
	require.Equal(t, http.StatusUnauthorized, readResponse.status, "unexpected http response status for insecure read")
}

func TestDenyReadExistingAttendee_Other(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given two users, the second of which has registered")
	token1 := tstValidUserToken(t, "101")
	location2, _ := tstRegisterAttendee(t, "ga2b-")

	docs.When("when the first one attempts to read the attendee info of the second one, i.e. of someone else")
	readResponse := tstPerformGet(location2, token1)

	docs.Then("then the request is denied as unauthorized (403) and the appropriate error is returned")
	tstRequireErrorResponse(t, readResponse, http.StatusForbidden, "auth.forbidden", "you are not authorized to access this data - the attempt has been logged")
}

func TestAllowReadExistingAttendee_Self(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an existing attendee who is logged in")
	token := tstValidUserToken(t, "101")
	location1, attendee1 := tstRegisterAttendeeWithToken(t, "ga3-", token)

	docs.When("when the first one attempts to read the attendee info of the second one")
	readResponse := tstPerformGet(location1, token)

	docs.Then("then the attendee is successfully read and the response is as expected")
	require.Equal(t, http.StatusOK, readResponse.status, "unexpected http response status")
	attendeeReadAgain := attendee.AttendeeDto{}
	tstParseJson(readResponse.body, &attendeeReadAgain)
	require.EqualValues(t, attendee1, attendeeReadAgain, "attendee data read did not match updated data")
}

func TestDenyReadExistingAttendeeWithStaffToken(t *testing.T) {
	docs.Given("given the configuration for staff pre-registration (the other config doesn't even have a staff token)")
	tstSetup(tstConfigFile(false, true, true))
	defer tstShutdown()

	docs.Given("given two existing users, the first of which is staff, and the second of which has registered")
	token := tstValidStaffToken(t, "202")
	location2, _ := tstRegisterAttendee(t, "ga4b-")

	docs.When("when the staffer attempts to read the attendee info of another user")
	readResponse := tstPerformGet(location2, token)

	docs.Then("then the request is denied as unauthorized (403) and the appropriate error is returned")
	tstRequireErrorResponse(t, readResponse, http.StatusForbidden, "auth.forbidden", "you are not authorized to access this data - the attempt has been logged")
}

func TestReadAttendeeNotFound(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an admin")
	token := tstValidAdminToken(t)

	docs.When("when they try to read an attendee that does not exist")
	response := tstPerformGet("/api/rest/v1/attendees/42", token)

	docs.Then("then the appropriate error response is returned")
	tstRequireErrorResponse(t, response, http.StatusNotFound, "attendee.id.notfound", "")
}

func TestReadAttendeeInvalidId(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	docs.Given("given an admin")
	token := tstValidAdminToken(t)

	docs.When("when they try to read an attendee with an invalid id")
	response := tstPerformGet("/api/rest/v1/attendees/smiling", token)

	docs.Then("then the appropriate error response is returned")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "attendee.id.invalid", "")
}

// --- attendee max id ---

func TestAttendeeMaxIdAvailable(t *testing.T) {
	docs.Given("given an existing attendee")
	tstSetup(tstConfigFile(false, false, true))
	defer tstShutdown()

	someAttendee := tstBuildValidAttendee("max1-")
	creationResponse := tstPerformPost("/api/rest/v1/attendees", tstRenderJson(someAttendee), tstNoToken())
	require.Equal(t, http.StatusCreated, creationResponse.status, "unexpected http response status for create")

	docs.When("when an unauthenticated user queries the maximum id")
	maxIdResponse := tstPerformGet("/api/rest/v1/attendees/max-id", tstNoToken())
	require.Equal(t, http.StatusOK, maxIdResponse.status, "unexpected http response status for max-id")

	docs.Then("then a positive number is returned")
	responseDto := attendee.AttendeeMaxIdDto{}
	tstParseJson(maxIdResponse.body, &responseDto)
	require.True(t, responseDto.MaxId > 0, "expected a positive number as maximum attendee id")
}

// helper functions

func tstReadAttendee(t *testing.T, location string) attendee.AttendeeDto {
	readAgainResponse := tstPerformGet(location, tstValidAdminToken(t))
	attendeeReadAgain := attendee.AttendeeDto{}
	tstParseJson(readAgainResponse.body, &attendeeReadAgain)
	return attendeeReadAgain
}
