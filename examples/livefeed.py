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
        # TODO: filter events
        app.config[TIME] = compare
        text = event + " @ " + str(compare)
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
    args = parser.parse_args()
    app.config[SERVER] = args.server
    app.config[PORT] = args.port
    app.config[SINCE] = None
    app.config[TIME] = None
    app.config[LAST] = None
    app.config[HISTORY] = []
    app.run()
