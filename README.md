Insider Back-end Task:

The deployed website Front-end can be accessed at the link: https://insider-back-end-task.onrender.com/ 
(It can take 1-2 mins for the cloud provider to start up after inactivity)

The codebase can be found at the GitHub repository: https://github.com/BDar01/Insider-Back-end-Task/tree/main

This is the SQL Schema I used via sqlite for the Insider Back-end Task,
consisting of two tables: teams and matches.

DROP TABLE IF EXISTS teams;
DROP TABLE IF EXISTS matches;

CREATE TABLE IF NOT EXISTS teams (
    id INTEGER PRIMARY KEY AUTOINCREMENT, -- Team ID
    name TEXT,                            -- Team name
    points INTEGER DEFAULT 0,             -- Points earned
    played INTEGER DEFAULT 0,             -- Matches played
    won INTEGER DEFAULT 0,                -- Matches won
    drawn INTEGER DEFAULT 0,              -- Matches drawn
    lost INTEGER DEFAULT 0,               -- Matches lost
    gf INTEGER DEFAULT 0,                 -- Goals for
    ga INTEGER DEFAULT 0,                 -- Goals against
    gd INTEGER DEFAULT 0,                 -- Goal difference
    strength INTEGER DEFAULT 1            -- Team strength
);

CREATE TABLE IF NOT EXISTS matches (
    id INTEGER PRIMARY KEY AUTOINCREMENT, -- Match ID
    home_team_id INTEGER,                 -- Home team ID
    away_team_id INTEGER,                 -- Away team ID
    home_score INTEGER,                   -- Home team score
    away_score INTEGER,                   -- Away team score
    week INTEGER                          -- Week of match
);

These are the SQL Queries used in the main.go file to read and update info in the database.

1. SeedDatabase function:
// Insert team strength into database
db.Exec("INSERT INTO teams (name, points, played, won, drawn, lost, gf, ga, gd, strength) VALUES (?, 0, 0, 0, 0, 0, 0, 0, 0, ?)", name, strength)

// Reset team stats to default values
db.Exec("UPDATE teams SET points = 0, played = 0, won = 0, drawn = 0, lost = 0, gf = 0, ga = 0, gd = 0 WHERE points IS NULL OR played IS NULL OR won IS NULL OR drawn IS NULL OR lost IS NULL OR gf IS NULL OR ga IS NULL OR gd IS NULL")

2. PlayWeekMatches function:
// Query to retrieve team data
rows, err := db.Query("SELECT id, name, strength FROM teams")

3. getPreviousWeekMatches function:
// Query to retrieve previous week matches
rows, err := db.Query("SELECT home_team_id, away_team_id FROM matches WHERE week = ?", week)

4. saveMatch function:
// Insert match data into matches table
db.Exec("INSERT INTO matches (home_team_id, away_team_id, home_score, away_score, week) VALUES (?, ?, ?, ?, ?)",
		match.HomeTeamID, match.AwayTeamID, match.HomeScore, match.AwayScore, match.Week)

5. updateTeamStats function:
// Retrieve current team stats
db.QueryRow("SELECT id, points, played, won, drawn, lost, gf, ga, gd, strength FROM teams WHERE id = ?", teamID).
		Scan(&team.ID, &team.Points, &team.Played, &team.Won, &team.Drawn, &team.Lost, &team.GF, &team.GA, &team.GD, &team.Strength)

// Update the team stats in the database
db.Exec("UPDATE teams SET points = ?, played = ?, won = ?, drawn = ?, lost = ?, gf = ?, ga = ?, gd = ?, strength = ? WHERE id = ?",
		team.Points, team.Played, team.Won, team.Drawn, team.Lost, team.GF, team.GA, team.GD, team.Strength, team.ID)

6. predictStandings function:
// Query to retrieve team data (points and GD) from the database
db.Query("SELECT id, name, points, gd FROM teams")

7. displayTableHTML function:
// Query to retrieve team stats where matches have been played
db.Query("SELECT name, points, played, won, drawn, lost, gd, strength FROM teams WHERE played > 0")

8. displayMatchResultsHTML function:
// Query to retrieve match results for the specified week
db.Query("SELECT home_team_id, away_team_id, home_score, away_score FROM matches WHERE week = ?", week)

9. getTeamName function:
// Query to get the team name from the database
db.QueryRow("SELECT name FROM teams WHERE id = ?", teamID).Scan(&name)
