# bets_crawler
crawler I built for EDUCATIONAL purpose

### Notes
- To use it just paste the url of the page with all dota 2 matches (on d2lounge or leonbets) into main.go and build it (Dockerfile will only run the executable)
- There are two more crawlers (for ggbet and another website) in core package, which I wont be fixing cuz ggbet does not provide services in russia anymore and the other website tries too hard to prevent ppl from parsing them
- Functions which crawl pages also write to db, so u'll have to change schema/persistence logic or figure out what is the schema of my db
