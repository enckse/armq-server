#!/usr/bin/python
"""live-feed streaming example."""

from flask import Flask
import json
import armq_workers as aw
import argparse

# config keys
SERVER = "server"
PORT = "port"
SINCE = "since"
TIME = "time"
LAST = "last"
HISTORY = "history"

EVENT_DATUM = {}
EVENT_DATUM["unit_killed"] = {"victim": "unit", "attacker": "unit"}
IGNORE_EVENTS = ["positions_infantry", "positions_vehicles"]

# flask
app = Flask(__name__)

# HTML formatting
CONTENT_KEY = "{content}"
HTML = """
<!doctype html>
<html>
<head>
<meta http-equiv="refresh" content="1" charset="UTF-8">
<style>
body
{
    background-color:#f0f0f0;
    font-family: "Helvetica Neue", Helvetica, Arial, sans-serif;
}

#main
{
    width: 70%;
    margin-left: auto;
    margin-right: auto;
    padding:20px 20px 20px 20px;
}
.entry
{
    border: 2px solid black;
    padding: 10px;
}

</style>
<title>livefeed</title>
</head>
<body>
<div id="main">
<h1>live feed</h1>
<hr />
""" + CONTENT_KEY + """
</div>
</body>
</html>
"""

ENTRY = "<div class=\"entry\">" + CONTENT_KEY + "</div>"


def _proc(key, data, time):
    """Process cache data."""
    for item in data[key]:
        parts = item.split("`")
        if not parts[0].endswith(":event"):
            continue
        compare = float(parts[-1])
        if compare == 0:
            continue
        if time is not None and compare <= time:
            continue
        event = parts[3]
        if event in EVENT_DATUM:
            datum = EVENT_DATUM[event]
            obj = json.loads(parts[4])
            event += "<hr>"
            added = []
            for item in datum:
                added.append("{} = '{}' ".format(item, obj[item][datum[item]]))
            event += "<br>".join(added)
        elif event in IGNORE_EVENTS:
            continue
        app.config[TIME] = compare
        text = event + "<hr>time: " + str(compare)
        hist = app.config[HISTORY]
        hist.insert(0, text)
        if len(hist) > 25:
            hist = hist[:-1]
        app.config[HISTORY] = hist
        html = "<br>".join([ENTRY.replace(CONTENT_KEY, x) for x in hist])
        app.config[LAST] = html
        return html
    return app.config[LAST]


@app.route('/')
def feed():
    """Get current feed."""
    data = _feed()
    datum = "no recent events"

    if data is not None:
        if len(data) > 0:
            for d in data:
                val = _proc(d, data, app.config[TIME])
                if val is not None:
                    datum = val
                    # NOTE: shortcut to start from the same, last received page
                    app.config[SINCE] = d - 1
                    break
    return HTML.replace(CONTENT_KEY, datum)


def _feed():
    """Request feed information."""
    req = aw.Request()
    req.since = app.config[SINCE]
    req.bucket = 100
    cached = aw.load_cached(app.config[SERVER], app.config[PORT], req)
    return cached


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('--port', type=int, default=6379)
    parser.add_argument('--server', type=str, default='localhost')
    parser.add_argument('--since', type=int, default=None)
    args = parser.parse_args()
    app.config[SERVER] = args.server
    app.config[PORT] = args.port
    app.config[SINCE] = args.since
    app.config[TIME] = None
    app.config[LAST] = None
    app.config[HISTORY] = []
    app.run()
