# bets_crawler
crawler I built for EDUCATIONAL purpose

### Notes
- To use it just paste the url of the page with all dota 2 matches (on d2lounge or leonbets) into main.go and build it (Dockerfile will only run the executable)
- There are two more crawlers (for ggbet and another website) in core package, which I wont be fixing cuz ggbet does not provide services in russia anymore and the other website tries too hard to prevent ppl from parsing them
- I write to db with no intermediate output, so u'll need a postgres instance (create a dotenv with DB_HOST, DB_PORT, DB_USER, DB_PASS and DB fields).
  - Schema:

    ```sql
    CREATE TABLE public.games (
        game_id integer NOT NULL,
        tournament character varying(250),
        radiant character varying(250),
        dire character varying(250),
        date timestamp with time zone
    );
    ALTER TABLE ONLY public.games
        ADD CONSTRAINT games_pkey PRIMARY KEY (game_id);
    ALTER TABLE ONLY public.games
        ADD CONSTRAINT games_radiant_date_key UNIQUE (radiant, date);
    ```
    ```sql
    CREATE TABLE public.bets (
        bet_id integer NOT NULL,
        type character varying(250),
        bet text[],
        game_id integer
    );
    ALTER TABLE ONLY public.bets
        ADD CONSTRAINT bets_pkey PRIMARY KEY (bet_id);
    ALTER TABLE ONLY public.bets
        ADD CONSTRAINT bets_game_id_fkey FOREIGN KEY (game_id) REFERENCES public.games(game_id);
    ```
