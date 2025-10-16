import sqlite3
import random
import time

DB_PATH = "web/blue_devil.db"  # Adjust path if needed

def inject_competition_scores(num_teams=3, num_rounds=5):
    conn = sqlite3.connect(DB_PATH)
    cur = conn.cursor()

    for round_num in range(1, num_rounds + 1):
        for team_id in range(1, num_teams + 1):
            score = random.randint(0, 10)
            desc = f"Injected score for team {team_id} round {round_num}"
            cur.execute(
                "INSERT INTO competition_scores (team_id, score, round, description, timestamp) VALUES (?, ?, ?, ?, ?)",
                (team_id, score, round_num, desc, time.strftime("%Y-%m-%d %H:%M:%S"))
            )
    conn.commit()
    conn.close()
    print(f"Injected {num_teams * num_rounds} competition_scores rows.")

def inject_competition_services(num_teams=3, num_services=3):
    conn = sqlite3.connect(DB_PATH)
    cur = conn.cursor()

    for team_id in range(1, num_teams + 1):
        for service_id in range(1, num_services + 1):
            status = random.choice(["up", "down", "degraded"])
            desc = f"Injected service status for team {team_id} service {service_id}"
            cur.execute(
                "INSERT INTO competition_services (team_id, service_id, status, description, timestamp) VALUES (?, ?, ?, ?, ?)",
                (team_id, service_id, status, desc, time.strftime("%Y-%m-%d %H:%M:%S"))
            )
    conn.commit()
    conn.close()
    print(f"Injected {num_teams * num_services} competition_services rows.")

if __name__ == "__main__":
    inject_competition_scores()
    inject_competition_services()