# insights-datasource-git
git data source V2

#### Environment Variables

- `GIT_STREAM` : name of the firehose stream to push the data to
- `AWS_REGION`,`AWS_ACCESS_KEY_ID` & `AWS_SECRET_ACCESS_KEY` : For AWS config to initialize firehose stream
#### Build & Run
- run `make` to build app.
- run `./scripts/example_run.sh` to try it.
- example [JSON](https://github.com/LF-Engineering/insights-datasource-git/blob/main/exampleOutput.json) generated by this tool:
```
[
  {
    "connector": "git",
    "connector_version": "0.1.1",
    "source": "git",
    "event_type": "commit.created",
    "created_by": "git-connector",
    "updated_by": "git-connector",
    "created_at": 1642591223,
    "updated_at": 1642591223,
    "payload": {
      "commit_id": "47ed92c89373caffac825bf5061171166dbfacd1f68cf336817d23b3174b5678",
      "sha": "a5dc87fbb1357518a9a1ea19a533977fbc74c819",
      "parent_shas": [
        "38165eecda3c01e17b47269e8a576103d8257efd"
      ],
      "repository_url": "https://github.com/lukaszgryglicki/trailers-test",
      "repository_id": "989987c88df0bc93ea433722e91c3bcb2de958582ff82ee0903bf846cc774a7a",
      "url": "https://github.com/lukaszgryglicki/trailers-test/commit/a5dc87fbb1357518a9a1ea19a533977fbc74c819",
      "message": "Missing dot\n\nSigned-off-by: Łukasz Gryglicki <lukaszgryglicki@o2.pl>\nAuthored-by: David Woodhouse <dwmw2@infradead.org> and Tilman Schmidt <tilman@imap.cc>\nAuthor: David Woodhouse <dwmw2@infradead.org> and Tilman Schmidt <tilman@imap.cc>\nCommit: David Woodhouse <dwmw2@infradead.org> and Tilman Schmidt <tilman@imap.cc>\nCo-authored-by: Andi Kleen <ak@suse.de>\nCo-authored-by: Andi2 Kleen2 <ak2@suse.de>",
      "contributors": [
        {
          "identity": {
            "identity_id": "3c3c043edad3943c293c985979f40201a4129c24",
            "email": "lukaszgryglicki@o2.pl",
            "name": "Łukasz Gryglicki",
            "source": "git"
          },
          "role": "author",
          "weight": 1
        },
        {
          "identity": {
            "identity_id": "3c3c043edad3943c293c985979f40201a4129c24",
            "email": "lukaszgryglicki@o2.pl",
            "name": "Łukasz Gryglicki",
            "source": "git"
          },
          "role": "committer",
          "weight": 1
        },
        {
          "identity": {
            "identity_id": "633882e7a48f37cb1af0a1bd304fd33d2450ff6f",
            "email": "ak@suse.de",
            "name": "Andi Kleen",
            "source": "git"
          },
          "role": "co_author",
          "weight": 1
        },
        {
          "identity": {
            "identity_id": "c536a1d36e7108171daa12b66a21f555def0e397",
            "email": "ak2@suse.de",
            "name": "Andi2 Kleen2",
            "source": "git"
          },
          "role": "co_author",
          "weight": 1
        },
        {
          "identity": {
            "identity_id": "3c3c043edad3943c293c985979f40201a4129c24",
            "email": "lukaszgryglicki@o2.pl",
            "name": "Łukasz Gryglicki",
            "source": "git"
          },
          "role": "signer",
          "weight": 1
        }
      ],
      "sync_timestamp": "2022-01-19T16:50:23.69555+05:30",
      "authored_timestamp": "2021-08-17T09:11:13Z",
      "committed_timestamp": "2021-08-17T09:11:17Z",
      "short_hash": "a5dc87",
      "files": [
        {
          "lines_added": 1,
          "lines_removed": 1,
          "type": "md",
          "files_modified": 1
        }
      ],
      "branch": "main",
      "default_branch": true
    }
  }
]
```
