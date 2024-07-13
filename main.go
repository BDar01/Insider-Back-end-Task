package main

import ( // Import required packages:
	"database/sql"  // For database operations
	"encoding/json" // For JSON encoding and decoding
	"fmt"           // For formatted I/O
	"log"           // For logging errors
	"math/rand"     // For generating random numbers
	"net/http"      // For HTTP server and request handling
	"sort"          // For sorting slices
	"strconv"       // For converting strings to integers
	"time"          // For time-related functions

	_ "modernc.org/sqlite" // SQLite driver (without CGO)
)

type Team struct { // Team represents a football team with its attributes
	ID       int    // Team ID
	Name     string // Team name
	Points   int    // Points earned
	Played   int    // Matches played
	Won      int    // Matches won
	Drawn    int    // Matches drawn
	Lost     int    // Matches lost
	GF       int    // Goals for
	GA       int    // Goals against
	GD       int    // Goal difference
	Strength int    // Team strength
}

type Match struct { // Match represents a football match played between two teams
	ID         int // Match ID
	HomeTeamID int // Home team ID
	AwayTeamID int // Away team ID
	HomeScore  int // Home team score
	AwayScore  int // Away team score
	Week       int // Week of match
}

type TeamPrediction struct { // TeamPrediction represents the predicted probability of a team winning the championship
	Name        string  // Team name
	Probability float64 // Probability of winning
}

func main() { // HTTP handlers for different routes on Front-end
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/simulate", simulateHandler)
	http.HandleFunc("/all", allLeagueHandler)
	http.HandleFunc("/changeStrengths", changeStrengthsHandler)
	http.HandleFunc("/teamStrengths", getTeamStrengthsHandler)

	db, err := SetupDatabase() // Initialize the database
	if err != nil {
		panic(err) // Return error if database fails to initialize
	}
	defer db.Close()

	SeedDatabase(db) // Seed the database with initial team data

	fmt.Println("Server active at http://localhost:8080/")
	log.Fatal(http.ListenAndServe(":8080", nil)) // Start the HTTP server
}

func indexHandler(w http.ResponseWriter, r *http.Request) { // indexHandler serves the main HTML Front-end file
	http.ServeFile(w, r, "index.html")
}

func simulateHandler(w http.ResponseWriter, r *http.Request) { // simulateHandler handles the simulation of 2 matches in a week
	weekStr := r.URL.Query().Get("week") // Retrieve relevant week from URL from Front-end query
	week, err := strconv.Atoi(weekStr)   // Convert week from string to int
	if err != nil {
		http.Error(w, "Invalid week parameter", http.StatusBadRequest) // Return error for invalid week
		return
	}

	db, err := sql.Open("sqlite", "file:league.db?cache=shared&mode=rwc&_loc=auto") // Open database via SQL
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError) // Return error if database fails to open
		return
	}
	defer db.Close() // Ensure database is closed by end of function

	PlayWeekMatches(db, week) // Simulate matches for the specified week

	// Generate HTML output for the results to display on Front-end
	output := fmt.Sprintf("<h2>%d%s Week</h2>\n", week, getOrdinalSuffix(week))
	output += "<h3>League Table</h3>\n"
	output += "<pre>\n"
	output += displayTableHTML(db)
	output += "</pre>\n"
	output += "<h3>Match Results</h3>\n"
	output += "<pre>\n"
	output += displayMatchResultsHTML(db, week)
	output += "</pre>\n"

	if week >= 4 { // Display predictions after week 4
		output += "<h3>Predictions for Championship</h3>\n"
		output += "<pre>\n"
		output += displayPredictionsHTML(db, week)
		output += "</pre>\n"
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, output)

	if week >= 5 { // Reset the database after week 5 for new simulation
		db, err = SetupDatabase() // Initialize the database
		if err != nil {
			http.Error(w, "Failed to reset database", http.StatusInternalServerError) // Return error if database fails to reset
			return
		}
		defer db.Close() // Ensure database is closed by end of function

		SeedDatabase(db) // Seed the database with initial team data
	}
}

func allLeagueHandler(w http.ResponseWriter, r *http.Request) { // allLeagueHandler handles the simulation of all weeks from current week to week 5
	db, err := sql.Open("sqlite", "file:league.db?cache=shared&mode=rwc&_loc=auto") // Open database via SQL
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError) // Return error if database fails to open
		return
	}
	defer db.Close()

	weekStr := r.URL.Query().Get("week")    // Retrieve relevant week from URL from Front-end query
	startWeek, err := strconv.Atoi(weekStr) // Convert week from string to int
	if err != nil || startWeek < 1 || startWeek > 5 {
		startWeek = 1 // Default to week 1 if week parameter is missing, invalid, or out of range
	}

	output := "" // Generate HTML output for the results to display on Front-end for remaining weeks
	for week := startWeek; week <= 5; week++ {
		PlayWeekMatches(db, week) // Simulate matches for the specified week

		output += fmt.Sprintf("<h2>%d%s Week</h2>\n", week, getOrdinalSuffix(week))
		output += "<h3>League Table</h3>\n"
		output += "<pre>\n"
		output += displayTableHTML(db)
		output += "</pre>\n"
		output += "<h3>Match Results</h3>\n"
		output += "<pre>\n"
		output += displayMatchResultsHTML(db, week)
		output += "</pre>\n"

		if week >= 4 { // Display predictions after week 4
			output += "<h3>Predictions for Championship</h3>\n"
			output += "<pre>\n"
			output += displayPredictionsHTML(db, week)
			output += "</pre>\n"
		}

		output += "<hr>\n"
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, output)

	// Reset the database after simulating all weeks
	db, err = SetupDatabase() // Initialize the database
	if err != nil {
		http.Error(w, "Failed to reset database", http.StatusInternalServerError) // Return error if database fails to reset
		return
	}
	defer db.Close() // Ensure database is closed by end of function

	SeedDatabase(db) // Seed the database with initial team data
}

func changeStrengthsHandler(w http.ResponseWriter, r *http.Request) { // changeStrengthsHandler handles update of team strengths
	var strengths map[string]int
	err := json.NewDecoder(r.Body).Decode(&strengths) // Parse the JSON body to get new team strengths
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest) // Return error for invalid strength
		return
	}

	db, err := sql.Open("sqlite", "file:league.db?cache=shared&mode=rwc&_loc=auto") // Open database via SQL
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError) // Return error if database fails to open
		return
	}
	defer db.Close() // Ensure database is closed by end of function

	stmt, err := db.Prepare("UPDATE teams SET strength = ? WHERE name = ?") // Prepare an update statement for changing team strengths
	if err != nil {
		http.Error(w, "Failed to prepare statement", http.StatusInternalServerError) // Return error if statement fails
		return
	}
	defer stmt.Close() // Ensure statement is closed by end of function

	for team, strength := range strengths { // Execute the update statement for each team
		if strength < 1 || strength > 4 {
			continue // Skip invalid strength values
		}
		_, err = stmt.Exec(strength, team)
		if err != nil {
			http.Error(w, "Failed to update team strength", http.StatusInternalServerError) // Return error if fail to update
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true}) // Respond with success
}

func getTeamStrengthsHandler(w http.ResponseWriter, r *http.Request) { // getTeamStrengthsHandler sends current team strengths to Front-end
	db, err := sql.Open("sqlite", "file:league.db?cache=shared&mode=rwc&_loc=auto") // Open database via SQL
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError) // Return error if database fails to open
		return
	}
	defer db.Close() // Ensure database is closed by end of function

	rows, err := db.Query("SELECT name, strength FROM teams") // Query to retrieve team strengths
	if err != nil {
		http.Error(w, "Failed to fetch team strengths", http.StatusInternalServerError) // Return error if query fails
		return
	}
	defer rows.Close() // Ensure rows are closed by end of function

	strengths := make(map[string]int) // Initialize map to hold team strengths
	for rows.Next() {
		var name string
		var strength int
		if err := rows.Scan(&name, &strength); err != nil {
			http.Error(w, "Failed to fetch team strengths", http.StatusInternalServerError) // Return error if row scan fails
			return
		}
		strengths[name] = strength // Add team strength to map
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Failed to fetch team strengths", http.StatusInternalServerError) // Return error if row processing fails
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(strengths); err != nil {
		http.Error(w, "Failed to encode team strengths", http.StatusInternalServerError) // Return error if JSON encoding fails
	}
}

func getOrdinalSuffix(n int) string { // getOrdinalSuffix returns the ordinal suffix (st, nd, rd, th) for each week number
	switch n % 10 { // Use switch statement to handle week n
	case 1:
		if n == 11 {
			return "th" // Check for 11th case
		}
		return "st" // Suffix for 1st
	case 2:
		if n == 12 {
			return "th" // Check for 12th case
		}
		return "nd" // Suffix for 2nd
	case 3:
		if n == 13 {
			return "th" // Check for 13th case
		}
		return "rd" // Suffix for 3rd
	default:
		return "th" // Suffix for remaining cases
	}
}

func SetupDatabase() (*sql.DB, error) { // SetupDatabase sets up the SQLite database with necessary tables
	db, err := sql.Open("sqlite", "file:league.db?cache=shared&mode=rwc&_loc=auto") // Open database via SQL
	if err != nil {
		return nil, err
	}

	dropTeamsTable := `DROP TABLE IF EXISTS teams;` // SQL statements to drop existing tables if any
	dropMatchesTable := `DROP TABLE IF EXISTS matches;`

	_, err = db.Exec(dropTeamsTable) // Execute DROP TABLE statement for teams
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(dropMatchesTable) // Execute DROP TABLE statement for matches
	if err != nil {
		return nil, err
	}

	// SQL statements to create new tables for teams and matches
	createTeamsTable := `CREATE TABLE IF NOT EXISTS teams (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT,
        points INTEGER DEFAULT 0,
        played INTEGER DEFAULT 0,
        won INTEGER DEFAULT 0,
        drawn INTEGER DEFAULT 0,
        lost INTEGER DEFAULT 0,
        gf INTEGER DEFAULT 0,
        ga INTEGER DEFAULT 0,
        gd INTEGER DEFAULT 0,
        strength INTEGER DEFAULT 1
    );`

	createMatchesTable := `CREATE TABLE IF NOT EXISTS matches (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        home_team_id INTEGER,
        away_team_id INTEGER,
        home_score INTEGER,
        away_score INTEGER,
        week INTEGER
    );`

	_, err = db.Exec(createTeamsTable) // Execute CREATE TABLE statement for teams
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(createMatchesTable) // Execute CREATE TABLE statement for matches
	if err != nil {
		return nil, err
	}

	return db, nil // Return initialized database
}

func SeedDatabase(db *sql.DB) { // SeedDatabase seeds the database with initial team data
	teams := []string{"Chelsea", "Arsenal", "Manchester City", "Liverpool"} // List of fixed team names
	rand.Seed(time.Now().UnixNano())                                        // Seed the random number generator
	for _, name := range teams {                                            // Initialize each team with random strength of 1-4
		strength := rand.Intn(4) + 1 // Random strength between 1 and 4
		// Insert team strength into database
		db.Exec("INSERT INTO teams (name, points, played, won, drawn, lost, gf, ga, gd, strength) VALUES (?, 0, 0, 0, 0, 0, 0, 0, 0, ?)", name, strength)
	}

	// Reset team stats to default values
	db.Exec("UPDATE teams SET points = 0, played = 0, won = 0, drawn = 0, lost = 0, gf = 0, ga = 0, gd = 0 WHERE points IS NULL OR played IS NULL OR won IS NULL OR drawn IS NULL OR lost IS NULL OR gf IS NULL OR ga IS NULL OR gd IS NULL")
}

func PlayWeekMatches(db *sql.DB, week int) { // PlayWeekMatches simulates matches for the given week
	var teams []Team
	rows, err := db.Query("SELECT id, name, strength FROM teams") // Query to retrieve team data
	if err != nil {
		panic(err) // Panic if query fails
	}
	defer rows.Close() // Ensure rows are closed by end of function

	for rows.Next() { // Iterate over each row of SQL query
		var team Team
		if err := rows.Scan(&team.ID, &team.Name, &team.Strength); err != nil {
			panic(err) // Panic if row scan fails
		}
		teams = append(teams, team) // Add team to list
	}

	if err := rows.Err(); err != nil {
		panic(err) // Panic if row processing fails
	}

	previousWeekMatches := getPreviousWeekMatches(db, week-1) // Get matches from the previous week

	rand.Seed(time.Now().UnixNano())                                                     // Seed the random number generator
	rand.Shuffle(len(teams), func(i, j int) { teams[i], teams[j] = teams[j], teams[i] }) // Shuffle teams' order initially

	for i := 0; i < len(teams); i += 2 { // Generate 2 matches per week
		if i+1 < len(teams) {
			for isRepeatMatch(previousWeekMatches, teams[i].ID, teams[i+1].ID) { // Check if possible match is a repeat
				rand.Shuffle(len(teams), func(i, j int) { teams[i], teams[j] = teams[j], teams[i] }) // Shuffle again if match is a repeat
			}
			playMatch(db, teams[i].ID, teams[i+1].ID, week, teams[i].Strength, teams[i+1].Strength) // Play match between two teams
		}
	}
}

func getPreviousWeekMatches(db *sql.DB, week int) []Match { // getPreviousWeekMatches returns the matches from the previous week
	var matches []Match
	rows, err := db.Query("SELECT home_team_id, away_team_id FROM matches WHERE week = ?", week) // Query to retrieve previous week matches
	if err != nil {
		panic(err) // Panic if query fails
	}
	defer rows.Close() // Ensure rows are closed by end of function

	for rows.Next() {
		var match Match
		if err := rows.Scan(&match.HomeTeamID, &match.AwayTeamID); err != nil {
			panic(err) // Panic if row scan fails
		}
		matches = append(matches, match) // Add match to list
	}

	if err := rows.Err(); err != nil {
		panic(err) // Panic if row processing fails
	}

	return matches
}

func isRepeatMatch(previousWeekMatches []Match, homeTeamID, awayTeamID int) bool { // isRepeatMatch checks if a match between two teams is a repeat of a previous week's match
	for _, match := range previousWeekMatches { // Check each match in previous week
		if (match.HomeTeamID == homeTeamID && match.AwayTeamID == awayTeamID) || (match.HomeTeamID == awayTeamID && match.AwayTeamID == homeTeamID) { // Check if team IDs match
			return true // Return true if match is a repeat
		}
	}
	return false // Return true if match isn't a repeat
}

func playMatch(db *sql.DB, homeTeamID, awayTeamID, week, homeStrength, awayStrength int) { // playMatch simulates a match between two teams and updates database with the result
	rand.Seed(time.Now().UnixNano()) // Seed the random number generator

	homeScore := 0 // Initialize team scores
	awayScore := 0

	if homeStrength > awayStrength { // Calculate score based on team strengths
		homeScore = rand.Intn(5)
		awayScore = rand.Intn(4)
		if awayScore > homeScore {
			awayScore, homeScore = homeScore, awayScore // Swap scores if mismatch
		}
	} else if awayStrength > homeStrength {
		homeScore = rand.Intn(4)
		awayScore = rand.Intn(5)
		if homeScore > awayScore {
			awayScore, homeScore = homeScore, awayScore // Swap scores if mismatch
		}
	} else { // When strengths are equal
		homeScore = rand.Intn(5)
		awayScore = rand.Intn(5)
	}

	if homeScore == 0 && awayScore == 0 { // Ensure at least one goal is scored if both are 0
		if homeStrength > awayStrength {
			homeScore = rand.Intn(4) + 1
		} else if homeStrength < awayStrength {
			awayScore = rand.Intn(4) + 1
		} else { // Equal chance if same strength
			homeScore = rand.Intn(4) + 1
			awayScore = rand.Intn(4) + 1
		}
	}

	match := Match{ // Initialize a Match object with values to be saved to database
		HomeTeamID: homeTeamID,
		AwayTeamID: awayTeamID,
		HomeScore:  homeScore,
		AwayScore:  awayScore,
		Week:       week,
	}

	saveMatch(db, match)
	updateLeagueTable(db, match)
}

func saveMatch(db *sql.DB, match Match) { // saveMatch saves a match result to the database
	_, err := db.Exec("INSERT INTO matches (home_team_id, away_team_id, home_score, away_score, week) VALUES (?, ?, ?, ?, ?)",
		match.HomeTeamID, match.AwayTeamID, match.HomeScore, match.AwayScore, match.Week) // Insert match data into matches table
	if err != nil {
		panic(err) // Panic if the query fails
	}
}

func updateLeagueTable(db *sql.DB, match Match) { // updateLeagueTable updates the league table for each team based on match result
	updateTeamStats(db, match.HomeTeamID, match.HomeScore, match.AwayScore) // Update stats for the home team
	updateTeamStats(db, match.AwayTeamID, match.AwayScore, match.HomeScore) // Update stats for the away team
}

func updateTeamStats(db *sql.DB, teamID, goalsFor, goalsAgainst int) { // updateTeamStats updates the stats of a team based on a match
	var team Team
	err := db.QueryRow("SELECT id, points, played, won, drawn, lost, gf, ga, gd, strength FROM teams WHERE id = ?", teamID).
		Scan(&team.ID, &team.Points, &team.Played, &team.Won, &team.Drawn, &team.Lost, &team.GF, &team.GA, &team.GD, &team.Strength) // Retrieve current team stats
	if err != nil {
		panic(err) // Panic if the query fails
	}

	team.Played++               // Increment number of matches played
	team.GF += goalsFor         // Update goals for of team
	team.GA += goalsAgainst     // Update goals against of team
	team.GD = team.GF - team.GA // Update goal difference of team

	if goalsFor > goalsAgainst { // Update match result in terms of points: won, drawn, and lost
		team.Won++
		team.Points += 3
	} else if goalsFor == goalsAgainst {
		team.Drawn++
		team.Points++
	} else {
		team.Lost++
	}

	_, err = db.Exec("UPDATE teams SET points = ?, played = ?, won = ?, drawn = ?, lost = ?, gf = ?, ga = ?, gd = ?, strength = ? WHERE id = ?",
		team.Points, team.Played, team.Won, team.Drawn, team.Lost, team.GF, team.GA, team.GD, team.Strength, team.ID) // Update the team stats in the database
	if err != nil {
		panic(err) // Panic if the team stats update fails
	}
}

func predictStandings(db *sql.DB) []TeamPrediction { // Predicts the standings of the teams based on their points and goal difference (GD)
	var teams []Team // Slice to store team data

	rows, err := db.Query("SELECT id, name, points, gd FROM teams") // Query to retrieve team data (points and GD) from the database
	if err != nil {
		panic(err) // Panic if the query fails
	}
	defer rows.Close() // Ensure the rows are closed after processing

	for rows.Next() { // Iterate through each row of the result
		var team Team
		if err := rows.Scan(&team.ID, &team.Name, &team.Points, &team.GD); err != nil {
			panic(err) // Panic if row scanning fails
		}
		teams = append(teams, team) // Add the team data to the slice
	}

	if err := rows.Err(); err != nil {
		panic(err) // Panic if row processing fails
	}

	sort.Slice(teams, func(i, j int) bool { // Sort the teams slice by points and GD
		if teams[i].Points == teams[j].Points {
			return teams[i].GD > teams[j].GD // Sort by GD in descending order when points are equal
		}
		return teams[i].Points > teams[j].Points // Else, sort by points in descending order
	})

	totalPoints := 0             // Total points of all teams
	totalPositiveGD := 0         // Total positive GD of all teams
	totalNegativeGD := 0         // Total negative GD of all teams (handle negative GD case)
	for _, team := range teams { // Iterate through each team to calculate total points and GD
		totalPoints += team.Points
		if team.GD >= 0 {
			totalPositiveGD += team.GD
		} else {
			totalNegativeGD += -team.GD // Convert negative GD to positive for summing
		}
	}

	var predictions []TeamPrediction
	for _, team := range teams { // Iterate through each team to calculate predictions
		var adjustedGD int // Adjusted GD based on +ve and -ve GD influence
		if team.GD >= 0 {
			adjustedGD = team.GD * totalPositiveGD / (totalPositiveGD + totalNegativeGD) // Adjust GD based on total positive GD if +ve
		} else {
			adjustedGD = -team.GD * totalNegativeGD / (totalPositiveGD + totalNegativeGD) // Adjust GD based on total negative GD if -ve
		}

		probability := float64(team.Points+adjustedGD) / float64(totalPoints) * 100 // Calculate probability including adjusted GD
		predictions = append(predictions, TeamPrediction{team.Name, probability})   // Add prediction to the slice
	}

	normalizeFactor := 0.0 // Factor to normalize probabilities
	for _, prediction := range predictions {
		normalizeFactor += prediction.Probability // Sum each prediction probability
	}
	for i := range predictions { // Normalize probabilities to ensure they sum up to 100%
		predictions[i].Probability = predictions[i].Probability * 100 / normalizeFactor
	}

	return predictions
}

func displayTableHTML(db *sql.DB) string { // Generates an HTML table displaying the league standings on Front-end
	output := "<div class=\"section-box\">\n" // Start the section box in HTML
	output += "<table>\n"                     // Start the table in HTML
	// Table header row
	output += "<tr><th>Team</th><th>PTS</th><th>P</th><th>W</th><th>D</th><th>L</th><th>GD</th><th>Str</th></tr>\n"

	rows, err := db.Query("SELECT name, points, played, won, drawn, lost, gd, strength FROM teams WHERE played > 0") // Query to retrieve team stats where matches have been played
	if err != nil {
		log.Println(err)
		return "" // Log error and return empty string if query fails
	}
	defer rows.Close() // Ensure rows are closed after processing

	for rows.Next() { // Iterate through each row of the query result
		var team Team
		if err := rows.Scan(&team.Name, &team.Points, &team.Played, &team.Won, &team.Drawn, &team.Lost, &team.GD, &team.Strength); err != nil {
			log.Println(err) // Log error if row scanning fails
			continue         // Continue to the next row if there is an error
		}
		output += fmt.Sprintf("<tr><td>%s</td><td>%d</td><td>%d</td><td>%d</td><td>%d</td><td>%d</td><td>%d</td><td>%d</td></tr>\n",
			team.Name, team.Points, team.Played, team.Won, team.Drawn, team.Lost, team.GD, team.Strength) // Add a row to the table with the team stats
	}

	output += "</table>\n" // End the table in HTML
	output += "</div>\n"   // End the section box in HTML

	if err := rows.Err(); err != nil {
		log.Println(err) // Log error if rrow processing fails
	}

	return output // Return the HTML output
}

func displayMatchResultsHTML(db *sql.DB, week int) string { // Generates an HTML section displaying match results for a specific week on Front-end

	output := "<div class=\"section-box\">"                                                // Start the section box in HTML
	output += fmt.Sprintf("<b>%d%s Week Match Result</b>\n", week, getOrdinalSuffix(week)) // Add the week title with suffix

	rows, err := db.Query("SELECT home_team_id, away_team_id, home_score, away_score FROM matches WHERE week = ?", week) // Query to retrieve match results for the specified week
	if err != nil {
		log.Println(err)
		return "" // Log error and return empty string if query fails
	}
	defer rows.Close() // Ensure rows are closed after processing

	for rows.Next() { // Iterate through each row of the query result
		var homeTeamID, awayTeamID, homeScore, awayScore int
		if err := rows.Scan(&homeTeamID, &awayTeamID, &homeScore, &awayScore); err != nil {
			log.Println(err) // Log error if row scanning fails
			continue         // Continue to next row if there is an error
		}
		homeTeamName := getTeamName(db, homeTeamID) // Get the home team name
		awayTeamName := getTeamName(db, awayTeamID) // Get the away team name
		// Add match result to the output
		output += fmt.Sprintf("%-20s %d - %-10d %-20s\n", homeTeamName, homeScore, awayScore, awayTeamName)
	}

	output += "</div>\n" // End the section box in HTML

	if err := rows.Err(); err != nil {
		log.Println(err) // Log error if row processing fails
	}

	return output // Return the HTML output
}

func displayPredictionsHTML(db *sql.DB, week int) string {
	output := "<div class=\"section-box\">\n"                                                              // Start the section box in HTML
	output += fmt.Sprintf("<b>%d%s Week Predictions for Championship</b>\n", week, getOrdinalSuffix(week)) // Add the week title with suffix

	predictions := predictStandings(db) // Calculate the predictions

	sort.Slice(predictions, func(i, j int) bool { // Sort predictions by probability in descending order
		return predictions[i].Probability > predictions[j].Probability
	})

	for idx, prediction := range predictions { // Iterate through each prediction to generate HTML output
		output += fmt.Sprintf("<b>%d.</b> %-20s %.2f<br>\n", idx+1, prediction.Name, prediction.Probability)
	}
	output += "</div>\n" // End the section box in HTML

	return output // Return the HTML output
}

func getTeamName(db *sql.DB, teamID int) string { // Retrieves the name of a team given its ID
	var name string
	err := db.QueryRow("SELECT name FROM teams WHERE id = ?", teamID).Scan(&name) // Query to get the team name from the database
	if err != nil {
		panic(err) // Panic if the query fails
	}
	return name
}
