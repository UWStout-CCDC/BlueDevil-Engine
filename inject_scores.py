import sqlite3
import random
import time
import argparse
import sys

DB_PATH = "web/blue_devil.db"  # Adjust path if needed

def inject_competition_services(num_teams=3, num_services=3):
    conn = sqlite3.connect(DB_PATH)
    cur = conn.cursor()

    def find_table_and_id_col(hint):
        cur.execute(
            "SELECT name FROM sqlite_master WHERE type='table' AND lower(name) LIKE ? ORDER BY name LIMIT 1",
            (f'%{hint}%',)
        )
        row = cur.fetchone()
        if not row:
            return None, None
        table = row[0]
        cur.execute(f"PRAGMA table_info('{table}')")
        cols = cur.fetchall()  # (cid, name, type, notnull, dflt_value, pk)
        # prefer 'id', then '*_id', then first integer column
        id_col = None
        for _, name, *_ in cols:
            if name.lower() == 'id':
                id_col = name
                break
        if not id_col:
            for _, name, *_ in cols:
                if name.lower().endswith('_id'):
                    id_col = name
                    break
        if not id_col:
            for _, name, ctype, *_ in cols:
                if ctype and 'int' in ctype.lower():
                    id_col = name
                    break
        return table, id_col

    # discover service ids
    s_table, s_col = find_table_and_id_col('service')
    if s_table and s_col:
        cur.execute(f"SELECT DISTINCT {s_col} FROM {s_table}")
        service_ids = [row[0] for row in cur.fetchall()]
    else:
        service_ids = list(range(1, num_services + 1))

    if not service_ids:
        service_ids = list(range(1, num_services + 1))

    # discover team ids
    t_table, t_col = find_table_and_id_col('team')
    if t_table and t_col:
        cur.execute(f"SELECT DISTINCT {t_col} FROM {t_table}")
        team_ids = [row[0] for row in cur.fetchall()]
    else:
        team_ids = list(range(1, num_teams + 1))

    if not team_ids:
        team_ids = list(range(1, num_teams + 1))

    # pick a random number for how many rounds to inject
    num_rounds = random.randint(1, 200)

    # For every round loop through all of the teams and services, and inject a random status
    for round_num in range(1, num_rounds + 1):
        for team_id in team_ids:
            for service_id in service_ids:
                status = random.choice(["up", "down"])
                desc = f"Injected service status for team {team_id} service {service_id} round {round_num}"
                cur.execute(
                    "INSERT INTO competition_services (team_id, service_id, is_up, output, round, timestamp) VALUES (?, ?, ?, ?, ?, ?)",
                    (team_id, service_id, status == "up", desc, round_num, time.strftime("%Y-%m-%d %H:%M:%S"))
                )
                # Calculate the score for each service status, if up +100, if down 0
                score = 100 if status == "up" else 0
                cur.execute(
                    "INSERT INTO competition_scores (team_id, score, round, description, timestamp) VALUES (?, ?, ?, ?, ?)",
                    (team_id, score, round_num, f"Score for team {team_id} service {service_id} round {round_num}", time.strftime("%Y-%m-%d %H:%M:%S"))
                )
    conn.commit()
    conn.close()
    print(f"Injected {len(team_ids) * len(service_ids) * num_rounds} competition_services rows using teams {team_ids} and services {service_ids}.")


def ensure_services(conn, desired=3):
    cur = conn.cursor()
    cur.execute("SELECT id, name FROM services ORDER BY id ASC")
    rows = cur.fetchall()
    existing = {r[1]: r[0] for r in rows}
    created = []
    # create missing services up to desired
    for i in range(1, desired + 1):
        name = f"Service-{i}"
        if name not in existing:
            cur.execute("INSERT INTO services (name, description) VALUES (?, ?)", (name, f"Auto-created {name}"))
            created_id = cur.lastrowid
            existing[name] = created_id
            created.append(created_id)
    conn.commit()
    # return all service ids (existing + created) as list sorted by id
    cur.execute("SELECT id FROM services ORDER BY id ASC")
    return [r[0] for r in cur.fetchall()]


def ensure_teams(conn, desired=3):
    cur = conn.cursor()
    cur.execute("SELECT id, name FROM teams ORDER BY id ASC")
    rows = cur.fetchall()
    existing = {r[1]: r[0] for r in rows}
    created = []
    for i in range(1, desired + 1):
        name = f"Team-{i}"
        if name not in existing:
            cur.execute("INSERT INTO teams (name) VALUES (?)", (name,))
            created_id = cur.lastrowid
            existing[name] = created_id
            created.append(created_id)
    conn.commit()
    cur.execute("SELECT id FROM teams ORDER BY id ASC")
    return [r[0] for r in cur.fetchall()]


def ensure_scored_boxes(conn, team_ids, service_ids, boxes_per_team=1, ip_base="10.0"):
    cur = conn.cursor()
    # For each team ensure at least boxes_per_team boxes exist; create new ones and attach a service_id in round-robin
    for idx, team_id in enumerate(team_ids, start=1):
        cur.execute("SELECT id FROM scored_boxes WHERE team_id = ? ORDER BY id ASC", (team_id,))
        existing = [r[0] for r in cur.fetchall()]
        need = boxes_per_team - len(existing)
        for n in range(need):
            ip = f"{ip_base}.{team_id % 255}.{(len(existing) + n) % 255}"
            # round-robin pick a service for this box
            svc = service_ids[(idx + n) % len(service_ids)] if service_ids else None
            cur.execute("INSERT INTO scored_boxes (team_id, ip_address, service_id) VALUES (?, ?, ?)", (team_id, ip, svc))
    conn.commit()


def run_auto_inject(num_teams=3, num_services=3, boxes_per_team=1):
    conn = sqlite3.connect(DB_PATH)
    try:
        services = ensure_services(conn, num_services)
        teams = ensure_teams(conn, num_teams)
        ensure_scored_boxes(conn, teams, services, boxes_per_team)
        print(f"Using services={services} teams={teams}")
    finally:
        conn.close()
    # Now inject competition rows using DB access inside inject_competition_services
    inject_competition_services(num_teams=num_teams, num_services=num_services)

def parse_args():
    p = argparse.ArgumentParser(description="Auto-create teams/services/boxes and inject competition data into the DB")
    p.add_argument("--teams", type=int, default=3, help="Number of teams to ensure/create")
    p.add_argument("--services", type=int, default=3, help="Number of services to ensure/create")
    p.add_argument("--boxes", type=int, default=1, help="Number of scored boxes per team to ensure/create")
    p.add_argument("--inject-only", action="store_true", help="Only inject competition data, do not create services/teams/boxes")
    p.add_argument("--seed", type=int, default=None, help="Optional random seed for reproducible injection")
    return p.parse_args()


if __name__ == "__main__":
    args = parse_args()
    if args.seed is not None:
        random.seed(args.seed)
    if args.inject_only:
        print("Injecting competition data only (no creation)")
        inject_competition_services(num_teams=args.teams, num_services=args.services)
        sys.exit(0)
    # run auto create then inject
    run_auto_inject(num_teams=args.teams, num_services=args.services, boxes_per_team=args.boxes)