package routes

import (
	// "net/http"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterInviteRoute(app core.App) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {

		se.Router.POST("/register", func(e *core.RequestEvent) error {

			// 1. Struct for incoming data
			data := struct {
				Email           string `json:"email"`
				Name        string `json:"name"`
				Password        string `json:"password"`
				PasswordConfirm string `json:"passwordConfirm"`
				Code            string `json:"code"`
			}{}

			// 2. Parse request body using v0.23+ helper
			if err := e.BindBody(&data); err != nil {
				return e.BadRequestError("Invalid request body", err)
			}

			// 3. Find the Invite Code (matching logic)
			// In v0.23+, we use app.FindRecordByFilter directly
			invite, err := app.FindFirstRecordByFilter("invites", "code={:code} && is_used=false", map[string]any{
				"code": data.Code,
			})

			if err != nil {
				return e.BadRequestError("Invalid invite code", err)
			}

			// 4. Prepare the User
			usersCollection, err := app.FindCollectionByNameOrId("users")
			if err != nil {
				return e.InternalServerError("Users collection not found", err)
			}

			newUser := core.NewRecord(usersCollection)
			newUser.Set("email", data.Email)
			newUser.Set("name", data.Name)
			newUser.Set("password", data.Password)
			newUser.Set("passwordConfirm", data.PasswordConfirm)

			// 5. Transaction: Create User + Mark Invite Used
			err = app.RunInTransaction(func(txApp core.App) error {
				if err := txApp.Save(newUser); err != nil {
					return err
				}

				invite.Set("is_used", true)
				invite.Set("used_by", newUser.Id)

				return txApp.Save(invite)
			})

			if err != nil {
				return e.BadRequestError("Failed to create account. Email may be taken.", err)
			}

			// 6. Return Auth Response
			// This helper generates the JSON response with token and user model
			return apis.RecordAuthResponse(e, newUser, "password", nil)
		})

		return se.Next()
	})
}
