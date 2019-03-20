#!/usr/bin/python3
"""Data extraction helper."""
import urllib.request
import argparse
import json
import os

_DATA = "data"


def _get_data(url):
    """Download data."""
    return urllib.request.urlopen(url).read().decode("utf-8")


def _get_tags(server, ranged):
    """Get tag list."""
    contents = _get_data(server + "/tags" + ranged)
    j = json.loads(contents)
    if _DATA not in j:
        raise Exception("invalid json response, no data")
    j = j[_DATA]
    for o in j:
        yield o


def _download(server, tag_storage, tagged):
    """Download a file."""
    url = server + "/?limit=0&filter=fields.tag.raw:eq:{}".format(tagged)
    content = _get_data(url)
    with open(tag_storage, 'w') as f:
        j = json.loads(content)
        f.write(json.dumps(j, indent=4))


def main():
    """Program entry."""
    parser = argparse.ArgumentParser()
    parser.add_argument("--server", default="http://localhost:8080")
    parser.add_argument("--cache", default="bin/")
    parser.add_argument("--query", default="")
    args = parser.parse_args()
    if not os.path.exists(args.cache):
        os.mkdir(args.cache)
    query = args.query
    if query:
        query = "?{}".format(query)
    print("downloading tags...")
    print("this may take a moment...")
    downloads = {}
    for t in _get_tags(args.server, query):
        print(t)
        for k in t.keys():
            print("found tag: {}".format(k))
            tag_file = os.path.join(args.cache) + "{}.json".format(k)
            if os.path.exists(tag_file):
                print(" -> already downloaded ({})".format(tag_file))
                continue
            downloads[k] = tag_file
    for d in downloads:
        print("downloading data: {}".format(d))
        _download(args.server, downloads[d], d)


if __name__ == "__main__":
    main()
