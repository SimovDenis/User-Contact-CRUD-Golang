package main

import (
	"database/sql"
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	//createDatabaseSchema()

	db := connectToDatabase()
	defer db.Close()

	a := app.New()
	w := a.NewWindow("Main Menu")
	w.Resize(fyne.NewSize(400, 250))

	txtEmail, txtPassword := widget.NewEntry(), widget.NewPasswordEntry()
	lblEmail, lblPassword := widget.NewLabel("Username/Email"), widget.NewLabel("Password")

	btnSignUp := createSignUpButton(a, db, txtEmail)
	btnSignIn := createSignInButton(db, txtEmail, txtPassword)

	w.SetContent(container.NewVBox(
		lblEmail,
		txtEmail,
		lblPassword,
		txtPassword,
		btnSignUp,
		btnSignIn,
	))

	w.CenterOnScreen()
	w.ShowAndRun()
}

func connectToDatabase() *sql.DB {
	db, err := sql.Open("mysql", "root:password@tcp(127.0.0.1:3307)/gaussdb")
	if err != nil {
		log.Fatal(err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	return db
}

func createSignUpButton(a fyne.App, db *sql.DB, txtEmail *widget.Entry) *widget.Button {
	return widget.NewButton("Sign up", func() {
		username := txtEmail.Text

		if checkUser(db, username) {
			fmt.Println("User already exists.")
		} else {
			fmt.Println("User does not exist.")
			showSignUpWindow(a, db)
		}
	})
}

func createSignInButton(db *sql.DB, txtEmail, txtPassword *widget.Entry) *widget.Button {
	return widget.NewButton("Sign in", func() {
		username, password := txtEmail.Text, txtPassword.Text

		valid, userID := checkCredentials(db, username, password)
		if valid {
			fmt.Println("Credentials are valid.")
			showContactsWindow(db, userID)
		} else {
			fmt.Println("Invalid credentials.")
		}
	})
}

func showSignUpWindow(a fyne.App, db *sql.DB) {
	win := a.NewWindow("Sign Up")
	win.Resize(fyne.NewSize(300, 250))

	usernameEntry := widget.NewEntry()
	emailEntry := widget.NewEntry()
	firstNameEntry := widget.NewEntry()
	lastNameEntry := widget.NewEntry()
	birthYearEntry := widget.NewEntry()
	passwordEntry := widget.NewPasswordEntry()
	oibEntry := widget.NewEntry()

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Username", Widget: usernameEntry},
			{Text: "Email", Widget: emailEntry},
			{Text: "First Name", Widget: firstNameEntry},
			{Text: "Last Name", Widget: lastNameEntry},
			{Text: "Birth Year", Widget: birthYearEntry},
			{Text: "Password", Widget: passwordEntry},
			{Text: "OIB", Widget: oibEntry},
		},
		OnSubmit: func() {
			addUser(db, a, win, usernameEntry.Text, emailEntry.Text, firstNameEntry.Text, lastNameEntry.Text, birthYearEntry.Text, passwordEntry.Text, oibEntry.Text)
			win.Close()
		},
	}

	win.SetContent(form)
	win.Show()
}

func addUser(db *sql.DB, a fyne.App, w fyne.Window, username, email, firstName, lastName, birthYear, password, oib string) {
	_, err := db.Exec("INSERT INTO user (username, email, first_name, last_name, birth_year, password, oib) VALUES (?, ?, ?, ?, ?, ?, ?)",
		username, email, firstName, lastName, birthYear, password, oib)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("User added successfully.")
}

func checkCredentials(db *sql.DB, username, password string) (bool, string) {
	var userID string

	query := "SELECT user_id FROM user WHERE (username=? OR email=?) AND password=?"
	err := db.QueryRow(query, username, username, password).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, ""
		} else {
			log.Fatal(err)
		}
	}

	return true, userID
}

func checkUser(db *sql.DB, username string) bool {
	var exists bool

	query := "SELECT EXISTS(SELECT 1 FROM user WHERE username=? OR email=?)"
	err := db.QueryRow(query, username, username).Scan(&exists)
	if err != nil {
		log.Fatal(err)
	}

	return exists
}

func showContactsWindow(db *sql.DB, username string) {
	a := app.New()
	win := a.NewWindow("Contacts")
	win.Resize(fyne.NewSize(800, 600))
	win.CenterOnScreen()

	contacts := getContacts(db, username)

	list := widget.NewList(
		func() int {
			return len(contacts)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			item.(*widget.Label).SetText(contacts[id])
		},
	)

	btnAddContact := widget.NewButton("Add Contact", func() {
		addContactWindow := a.NewWindow("Add Contact")
		addContactWindow.Resize(fyne.NewSize(400, 300))

		firstNameEntry := widget.NewEntry()
		lastNameEntry := widget.NewEntry()
		contactNumberEntry := widget.NewEntry()
		emailEntry := widget.NewEntry()

		btnSubmit := widget.NewButton("Submit", func() {
			firstName := firstNameEntry.Text
			lastName := lastNameEntry.Text
			contactNumber := contactNumberEntry.Text
			email := emailEntry.Text

			addContact(db, username, firstName, lastName, contactNumber, email)

			contacts = getContacts(db, username)
			list.Refresh()

			addContactWindow.Close()
		})

		addContactWindow.SetContent(container.NewVBox(
			widget.NewLabel("First Name"),
			firstNameEntry,
			widget.NewLabel("Last Name"),
			lastNameEntry,
			widget.NewLabel("Contact Number"),
			contactNumberEntry,
			widget.NewLabel("Email"),
			emailEntry,
			btnSubmit,
		))

		addContactWindow.Show()
	})

	content := container.New(layout.NewVBoxLayout(), list, btnAddContact)
	win.SetContent(content)
	win.Show()
}

func getContacts(db *sql.DB, userID string) []string {
	rows, err := db.Query(
		"SELECT c.first_name, c.last_name, c.contact_number FROM user_contact uc JOIN contact c ON uc.contact_id = c.user_id WHERE uc.user_id = ?", userID)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var contacts []string
	for rows.Next() {
		var firstName, lastName, contactNumber string
		if err := rows.Scan(&firstName, &lastName, &contactNumber); err != nil {
			log.Fatal(err)
		}
		contacts = append(contacts, firstName+" "+lastName+": "+contactNumber)
	}

	return contacts
}

func addContact(db *sql.DB, userID, firstName, lastName, contactNumber, email string) {

	_, err := db.Exec(
		"INSERT INTO contact (first_name, last_name, contact_number, email) VALUES (?, ?, ?, ?)",
		firstName, lastName, contactNumber, email)
	if err != nil {
		log.Fatal(err)
	}

	var contactID string
	err = db.QueryRow("SELECT LAST_INSERT_ID()").Scan(&contactID)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(
		"INSERT INTO user_contact (user_id, contact_id) VALUES (?, ?)",
		userID, contactID)
	if err != nil {
		log.Fatal(err)
	}
}

/* func promptUserForFirstName(win fyne.Window) string {
	return promptUserForInput(win, "First Name")
}

func promptUserForLastName(win fyne.Window) string {
	return promptUserForInput(win, "Last Name")
}

func promptUserForContactNumber(win fyne.Window) string {
	return promptUserForInput(win, "Contact Number")
}

func promptUserForEmail(win fyne.Window) string {
	return promptUserForInput(win, "Email")
}

func promptUserForInput(win fyne.Window, prompt string) string {
	entry := widget.NewEntry()
	dialog.ShowForm(prompt, "OK", "Cancel", []*widget.FormItem{
		{Text: prompt, Widget: entry},
	}, func(ok bool) {
		if !ok {
			return
		}
	}, win)
	return entry.Text
} */
