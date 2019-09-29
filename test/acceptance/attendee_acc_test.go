package acceptance

import (
	"github.com/jumpy-squirrel/rexis-go-attendee/internal/repository/config"
	"github.com/stretchr/testify/require"
	"net/http"
	"github.com/jumpy-squirrel/rexis-go-attendee/api/v1/attendee"
	"github.com/jumpy-squirrel/rexis-go-attendee/docs"
	"testing"
)

// ------------------------------------------
// acceptance tests for the attendee resource
// ------------------------------------------

// --- create new attendee ---

func TestCreateNewAttendee(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstDefaultConfigFile)
	defer tstShutdown()

	docs.Given("given an unauthenticated user")

	docs.When( "when they create a new attendee with valid data")
	attendeeSent := tstBuildValidAttendee()
	response := tstPerformPut("/api/rest/v1/attendees", tstRenderJson(attendeeSent), tstNoToken())

	docs.Then( "then the attendee is successfully created")
	require.Equal(t, http.StatusCreated, response.status, "unexpected http response status")
	require.Regexp(t, "^\\/api\\/rest\\/v1\\/attendees\\/[1-9][0-9]*$", response.location, "invalid location header in response")
}

func TestCreateNewAttendeeInvalid(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstDefaultConfigFile)
	defer tstShutdown()

	docs.Given("given an unauthenticated user")

	docs.When( "when they create a new attendee with invalid data")
	attendeeSent := tstBuildValidAttendee()
	attendeeSent.Nickname = "$%&^@!$"
	attendeeSent.Packages = attendeeSent.Packages + ",sponsor" // a constraint violation
	attendeeSent.Birthday = "2004-11-23" // too young
	response := tstPerformPut("/api/rest/v1/attendees", tstRenderJson(attendeeSent), tstNoToken())

	docs.Then( "then the attendee is rejected with an appropriate error response")
	require.Equal(t, http.StatusBadRequest, response.status, "unexpected http response status")
	errorDto := attendee.ErrorDto{}
	tstParseJson(response.body, &errorDto)
	require.Equal(t, "attendee.data.invalid", errorDto.Message, "unexpected error code")
	require.Equal(t, "nickname field must contain at least two letters, and contain no more than two non-letters", errorDto.Details.Get("nickname"))
	require.Equal(t, "cannot pick both sponsor2 and sponsor - constraint violated", errorDto.Details.Get("packages"))
	require.Equal(t, "birthday must be no earlier than 1901-01-01 and no later than 2001-08-14", errorDto.Details.Get("birthday"))
}

func TestCreateNewAttendeeCanBeReadAgainByAdmin(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstDefaultConfigFile)
	defer tstShutdown()

	docs.Given("given an unauthenticated user")

	docs.When( "when they create a new attendee")
	attendeeSent := tstBuildValidAttendee()
	response := tstPerformPut("/api/rest/v1/attendees", tstRenderJson(attendeeSent), tstNoToken())

	docs.Then( "then the attendee is successfully created and its data can be read again by an admin")
	require.Equal(t, http.StatusCreated, response.status, "unexpected http response status")
	require.Regexp(t, "^\\/api\\/rest\\/v1\\/attendees\\/[1-9][0-9]*$", response.location, "invalid location header in response")

	attendeeReadAgain := tstReadAttendee(t, response.location)
	// difference in id is ok, so copy it over to expected
	attendeeSent.Id = attendeeReadAgain.Id
	require.EqualValues(t, attendeeSent, attendeeReadAgain, "attendee data read did not match sent data")
}

func TestCreateNewAttendeeStaffregNotLoggedIn(t *testing.T) {
	docs.Given("given the configuration for staff pre-registration")
	tstSetup(tstStaffregConfigFile)
	defer tstShutdown()

	docs.Given("given an unauthenticated user")

	docs.When( "when they attempt to create a new attendee with valid data")
	attendeeSent := tstBuildValidAttendee()
	response := tstPerformPut("/api/rest/v1/attendees", tstRenderJson(attendeeSent), tstNoToken())

	docs.Then( "then the request is denied as unauthenticated (401) and no location header is supplied")
	require.Equal(t, http.StatusUnauthorized, response.status, "unexpected http response status")
	require.Equal(t, "", response.location, "non-empty location header in response")
}

func TestCreateNewAttendeeStaffregStaff(t *testing.T) {
	docs.Given("given the configuration for staff pre-registration")
	tstSetup(tstStaffregConfigFile)
	defer tstShutdown()

	docs.Given("given a staffer")
	staffToken := tstValidStaffToken(t)

	docs.When( "when they attempt to create a new attendee with valid data")
	attendeeSent := tstBuildValidAttendee()
	response := tstPerformPut("/api/rest/v1/attendees", tstRenderJson(attendeeSent), staffToken)

	docs.Then( "then the attendee is successfully created")
	require.Equal(t, http.StatusCreated, response.status, "unexpected http response status")
	require.Regexp(t, "^\\/api\\/rest\\/v1\\/attendees\\/[1-9][0-9]*$", response.location, "invalid location header in response")
}

func TestCreateNewAttendeeStaffregUser(t *testing.T) {
	docs.Given("given the configuration for staff pre-registration")
	tstSetup(tstStaffregConfigFile)
	defer tstShutdown()

	docs.Given("given an authenticated regular user")
	userToken := tstValidUserToken(t)

	docs.When( "when they attempt to create a new attendee with valid data")
	attendeeSent := tstBuildValidAttendee()
	response := tstPerformPut("/api/rest/v1/attendees", tstRenderJson(attendeeSent), userToken)

	docs.Then( "then the request is denied as unauthorized (403) and no location header is supplied")
	require.Equal(t, http.StatusForbidden, response.status, "unexpected http response status")
	require.Equal(t, "", response.location, "non-empty location header in response")
}

func TestCreateNewAttendeeStaffregAdmin(t *testing.T) {
	docs.Given("given the configuration for staff pre-registration")
	tstSetup(tstStaffregConfigFile)
	defer tstShutdown()

	docs.Given("given an admin")
	adminToken := tstValidAdminToken(t)

	docs.When( "when they attempt to create a new attendee with valid data")
	attendeeSent := tstBuildValidAttendee()
	response := tstPerformPut("/api/rest/v1/attendees", tstRenderJson(attendeeSent), adminToken)

	docs.Then( "then the attendee is successfully created")
	require.Equal(t, http.StatusCreated, response.status, "unexpected http response status")
	require.Regexp(t, "^\\/api\\/rest\\/v1\\/attendees\\/[1-9][0-9]*$", response.location, "invalid location header in response")
}

func TestCreateNewAttendeeAdminOnlyFlag(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstDefaultConfigFile)
	defer tstShutdown()

	docs.Given("given an unauthenticated user")

	docs.When( "when they send a new attendee and attempt to set an admin only flag (guest)")
	attendeeSent := tstBuildValidAttendee()
	attendeeSent.Flags = "guest,hc"
	response := tstPerformPut("/api/rest/v1/attendees", tstRenderJson(attendeeSent), tstNoToken())

	docs.Then( "then the attendee is rejected with an error response")
	require.Equal(t, http.StatusBadRequest, response.status, "unexpected http response status")
	errorDto := attendee.ErrorDto{}
	tstParseJson(response.body, &errorDto)
	require.Equal(t, "attendee.data.invalid", errorDto.Message, "unexpected error code")
}

func TestCreateNewAttendeeDefaultAdminOnlyPackage(t *testing.T) {
	docs.Given("given the configuration for staff pre-registration")
	tstSetup(tstStaffregConfigFile)
	defer tstShutdown()

	docs.Given("given a staffer")
	staffToken := tstValidStaffToken(t)

	docs.When( "when they send a new attendee and attempt to leave out an admin only default package (room-none)")
	attendeeSent := tstBuildValidAttendee()
	attendeeSent.Packages = "attendance,stage,sponsor"
	response := tstPerformPut("/api/rest/v1/attendees", tstRenderJson(attendeeSent), staffToken)

	docs.Then( "then the attendee is rejected with an error response")
	require.Equal(t, http.StatusBadRequest, response.status, "unexpected http response status")
	errorDto := attendee.ErrorDto{}
	tstParseJson(response.body, &errorDto)
	require.Equal(t, "attendee.data.invalid", errorDto.Message, "unexpected error code")
}

// --- update attendee ---

func TestUpdateExistingAttendee(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstDefaultConfigFile)
	defer tstShutdown()

	docs.Given("given an existing attendee")
	existingAttendee := tstBuildValidAttendee()
	creationResponse := tstPerformPut("/api/rest/v1/attendees", tstRenderJson(existingAttendee), tstNoToken())
	require.Equal(t, http.StatusCreated, creationResponse.status, "unexpected http response status for create")
	attendeeReadAfterCreation := tstReadAttendee(t, creationResponse.location)

	docs.When( "when they send updated attendee info while logged in")
	docs.Limitation("the current fixed-token security model cannot check which user is logged in. This is ok because only the old regsys will know the user / admin tokens.")
	changedAttendee := attendeeReadAfterCreation
	changedAttendee.FirstName = "Eva"
	changedAttendee.LastName = "Musterfrau"
	updateResponse := tstPerformPost(creationResponse.location, tstRenderJson(changedAttendee), tstValidUserToken(t))

	docs.Then( "then the attendee is successfully updated and the changed data can be read again")
	require.Equal(t, http.StatusOK, updateResponse.status, "unexpected http response status for update")
	require.Equal(t, creationResponse.location, updateResponse.location, "location unexpectedly changed during update")
	attendeeReadAgain := tstReadAttendee(t, creationResponse.location)
	require.EqualValues(t, changedAttendee, attendeeReadAgain, "attendee data read did not match updated data")
}

func TestDenyUpdateExistingAttendeeWhileNotLoggedIn(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstDefaultConfigFile)
	defer tstShutdown()

	docs.Given("given an existing attendee and a user who is not logged in")
	existingAttendee := tstBuildValidAttendee()
	existingAttendee.FirstName = "Marianne"
	creationResponse := tstPerformPut("/api/rest/v1/attendees", tstRenderJson(existingAttendee), tstNoToken())
	require.Equal(t, http.StatusCreated, creationResponse.status, "unexpected http response status for create")
	attendeeReadAfterCreation := tstReadAttendee(t, creationResponse.location)

	docs.When( "when they send updated attendee info while not logged in")
	changedAttendee := attendeeReadAfterCreation
	changedAttendee.FirstName = "Eva"
	updateResponse := tstPerformPost(creationResponse.location, tstRenderJson(changedAttendee), tstNoToken())

	docs.Then( "then the request is denied as unauthenticated (401) and the data remains unchanged")
	require.Equal(t, http.StatusUnauthorized, updateResponse.status, "unexpected http response status for insecure update")
	attendeeReadAgain := tstReadAttendee(t, creationResponse.location)
	require.EqualValues(t, "Marianne", attendeeReadAgain.FirstName, "attendee data read did not match original data")
}

func TestDenyUpdateExistingAttendeeWithStaffToken(t *testing.T) {
	docs.Given("given the configuration for staff pre-registration (the other config doesn't even have a staff token)")
	tstSetup(tstStaffregConfigFile)
	defer tstShutdown()

	docs.Given("given an existing attendee")
	existingAttendee := tstBuildValidAttendee()
	existingAttendee.FirstName = "Marianne"
	creationResponse := tstPerformPut("/api/rest/v1/attendees", tstRenderJson(existingAttendee), tstValidAdminToken(t))
	require.Equal(t, http.StatusCreated, creationResponse.status, "unexpected http response status for create")
	attendeeReadAfterCreation := tstReadAttendee(t, creationResponse.location)

	docs.When( "when a logged in staffer sends updated attendee info")
	changedAttendee := attendeeReadAfterCreation
	changedAttendee.FirstName = "Eva"
	updateResponse := tstPerformPost(creationResponse.location, tstRenderJson(changedAttendee), tstValidStaffToken(t))

	docs.Then( "then the request is denied as unauthorized (403) and the data remains unchanged")
	require.Equal(t, http.StatusForbidden, updateResponse.status, "unexpected http response status for insecure update")
	attendeeReadAgain := tstReadAttendee(t, creationResponse.location)
	require.EqualValues(t, "Marianne", attendeeReadAgain.FirstName, "attendee data read did not match original data")
}

func TestUpdateExistingAttendeeAdminOnlyFlag(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstDefaultConfigFile)
	defer tstShutdown()

	docs.Given("given an existing attendee")
	existingAttendee := tstBuildValidAttendee()
	creationResponse := tstPerformPut("/api/rest/v1/attendees", tstRenderJson(existingAttendee), tstNoToken())
	require.Equal(t, http.StatusCreated, creationResponse.status, "unexpected http response status for create")
	attendeeReadAfterCreation := tstReadAttendee(t, creationResponse.location)

	docs.When( "when they send updated attendee info and attempt to add an admin-only flag (guest)")
	changedAttendee := attendeeReadAfterCreation
	changedAttendee.Flags = changedAttendee.Flags + ",guest"
	updateResponse := tstPerformPost(creationResponse.location, tstRenderJson(changedAttendee), tstValidUserToken(t))

	docs.Then( "then the request is denied and the data remains unchanged")
	require.Equal(t, http.StatusBadRequest, updateResponse.status, "unexpected http response status for malicious update")
	attendeeReadAgain := tstReadAttendee(t, creationResponse.location)
	require.EqualValues(t, "anon,ev", attendeeReadAgain.Flags, "attendee data read did not match original data")
}

// --- get attendee ---

func TestDenyReadExistingAttendeeWhileNotLoggedIn(t *testing.T) {
	docs.Given("given the configuration for standard registration")
	tstSetup(tstDefaultConfigFile)
	defer tstShutdown()

	docs.Given("given an existing attendee and a user who is not logged in")
	existingAttendee := tstBuildValidAttendee()
	creationResponse := tstPerformPut("/api/rest/v1/attendees", tstRenderJson(existingAttendee), tstNoToken())
	require.Equal(t, http.StatusCreated, creationResponse.status, "unexpected http response status for create")

	docs.When( "when they attempt to read attendee info while not logged in")
	readResponse := tstPerformGet(creationResponse.location, tstNoToken())

	docs.Then( "then the request is denied")
	require.Equal(t, http.StatusUnauthorized, readResponse.status, "unexpected http response status for insecure read")
}

func TestDenyReadExistingAttendeeWithStaffToken(t *testing.T) {
	docs.Given("given the configuration for staff pre-registration (the other config doesn't even have a staff token)")
	tstSetup(tstStaffregConfigFile)
	defer tstShutdown()

	docs.Given("given an existing attendee")
	existingAttendee := tstBuildValidAttendee()
	creationResponse := tstPerformPut("/api/rest/v1/attendees", tstRenderJson(existingAttendee), tstValidAdminToken(t))
	require.Equal(t, http.StatusCreated, creationResponse.status, "unexpected http response status for create")

	docs.When( "when a logged in staffer attempts to read the attendee info")
	readResponse := tstPerformGet(creationResponse.location, tstValidStaffToken(t))

	docs.Then( "then the request is denied as unauthorized (403)")
	require.Equal(t, http.StatusForbidden, readResponse.status, "unexpected http response status for insecure read")
}

// helper functions

func tstReadAttendee(t *testing.T, location string) attendee.AttendeeDto {
	readAgainResponse := tstPerformGet(location, tstValidAdminToken(t))
	attendeeReadAgain := attendee.AttendeeDto{}
	tstParseJson(readAgainResponse.body, &attendeeReadAgain)
	return attendeeReadAgain
}

func tstNoToken() string {
	return ""
}

func tstValidUserToken(t *testing.T) string {
	token, err := config.FixedToken(config.TokenForLoggedInUser)
	require.Nil(t, err)
	return token
}

func tstValidAdminToken(t *testing.T) string {
	token, err := config.FixedToken(config.TokenForAdmin)
	require.Nil(t, err)
	return token
}

func tstValidStaffToken(t *testing.T) string {
	token, err := config.FixedToken(config.OptionalTokenForInitialReg)
	require.Nil(t, err)
	require.NotEqual(t, "", token)
	return token
}
