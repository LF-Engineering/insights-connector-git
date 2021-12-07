# insights-datasource-git
git data source V2

#### Environment Variables

- `DA_GIT_STREAM` : name of the firehose stream to push the data to
- `AWS_REGION`,`AWS_ACCESS_KEY_ID` & `AWS_SECRET_ACCESS_KEY` : For AWS config to initialize firehose stream
#### Build & Run
- run `make swagger` to generate models.
- run `make` to build app.
- run `./scripts/example_run.sh` to try it.
- example [JSON](https://github.com/LF-Engineering/insights-datasource-git/blob/main/exampleOutput.json) generated by this tool:
```
{
  "DataSource": {
    "Name": "git",
    "Slug": "git"
  },
  "Endpoint": "https://github.com/lukaszgryglicki/trailers-test",
  "Events": [
    {
      "Commit": {
        "CommitURL": "https://github.com/lukaszgryglicki/trailers-test/commit/0e5e87e7fcda7b6f133fb969d88ae8d0c68e3533",
        "DataSourceId": "git",
        "Files": [
          {
            "Action": "A",
            "Added": 15,
            "Changed": 15,
            "Name": ".gitignore"
          },
          {
            "Action": "A",
            "Added": 201,
            "Changed": 201,
            "Name": "LICENSE"
          },
          {
            "Action": "A",
            "Added": 2,
            "Changed": 2,
            "Name": "README.md"
          }
        ],
        "HashShort": "0e5e87",
        "Id": "4bb86c9ccb2e0394b3bc0b1cb9c6b2b179aebc28",
        "IsDoc": true,
        "Message": "Initial commit",
        "Parents": [],
        "RepositoryShortURL": "trailers-test",
        "RepositoryType": "github",
        "RepositoryURL": "https://github.com/lukaszgryglicki/trailers-test",
        "Roles": [
          {
            "Id": "9dc6a328c1512ddb46078377e599b0e89a5bbf35",
            "Name": "author",
            "User": {
              "DataSourceId": "git",
              "Id": "aaa8024197795de9b90676592772633c5cfcb35a",
              "Name": "Łukasz Gryglicki"
            },
            "Weight": 1,
            "When": "2021-03-25T07:50:52.000Z",
            "WhenInTz": "2021-03-25T08:50:52.000Z",
            "WhenTz": 1
          },
          {
            "Id": "00ed13a2ba2c2a949530927f093b145822d50e8d",
            "Name": "committer",
            "User": {
              "DataSourceId": "git",
              "Id": "c780b9e5b86174118a9dbc1b738e9cd0e5fa3566",
              "Name": "GitHub"
            },
            "Weight": 1,
            "When": "2021-03-25T07:50:52.000Z",
            "WhenInTz": "2021-03-25T08:50:52.000Z",
            "WhenTz": 1
          }
        ],
        "SHA": "0e5e87e7fcda7b6f133fb969d88ae8d0c68e3533",
        "Stats": {
          "LinesAdded": 218,
          "LinesChanged": 218
        },
        "Title": "Initial commit"
      }
    },
    {
      "Commit": {
        "CommitURL": "https://github.com/lukaszgryglicki/trailers-test/commit/ec71bbb88cb677f8c5fbaf6962ade34c1348ed73",
        "DataSourceId": "git",
        "Files": [],
        "HashShort": "ec71bb",
        "Id": "eb97d6e443f8b8d84f8b5f4c09fb2d1eccdbec51",
        "Message": "Merge branch 'main' of https://github.com/lukaszgryglicki/trailers-test",
        "Parents": [
          "0feccac2c7ae7dfb272a23d76c4cd471ab42b57d",
          "0012369253cf55c4c026229f629444abec9cf4e5"
        ],
        "RepositoryShortURL": "trailers-test",
        "RepositoryType": "github",
        "RepositoryURL": "https://github.com/lukaszgryglicki/trailers-test",
        "Roles": [
          {
            "Id": "83d61f5bfbbb60f0c66a87db95d3bf3a5f3af619",
            "Name": "author",
            "User": {
              "DataSourceId": "git",
              "Id": "582d5784c59885c497f615bd692bc031c42a18cc",
              "Name": "Łukasz Gryglicki"
            },
            "Weight": 1,
            "When": "2021-03-25T09:08:58.000Z",
            "WhenInTz": "2021-03-25T09:08:58.000Z"
          },
          {
            "Id": "58d9c68e839e300a90fd8168f1dc71357fa7e755",
            "Name": "committer",
            "User": {
              "DataSourceId": "git",
              "Id": "582d5784c59885c497f615bd692bc031c42a18cc",
              "Name": "Łukasz Gryglicki"
            },
            "Weight": 1,
            "When": "2021-03-25T09:08:58.000Z",
            "WhenInTz": "2021-03-25T09:08:58.000Z"
          }
        ],
        "SHA": "ec71bbb88cb677f8c5fbaf6962ade34c1348ed73",
        "Stats": {},
        "Title": "Merge branch 'main' of https://github.com/lukaszgryglicki/trailers-test"
      }
    },
    {
      "Commit": {
        "CommitURL": "https://github.com/lukaszgryglicki/trailers-test/commit/0feccac2c7ae7dfb272a23d76c4cd471ab42b57d",
        "DataSourceId": "git",
        "Files": [
          {
            "Action": "M",
            "Added": 1,
            "Changed": 1,
            "Name": "file.txt"
          }
        ],
        "HashShort": "0fecca",
        "Id": "8b40a7d360ebe53f2298128c279f13d7f7c5d97e",
        "IsDoc": true,
        "Message": "test PP\n\nSigned-off-by: Łukasz Gryglicki <lukaszgryglicki@o2.pl>\nCo-authored-by: Unicron <unicron@multiverse.uv>",
        "Parents": [
          "d465d4c2c75e6cb36fb0596aede5364713586cf2"
        ],
        "RepositoryShortURL": "trailers-test",
        "RepositoryType": "github",
        "RepositoryURL": "https://github.com/lukaszgryglicki/trailers-test",
        "Roles": [
          {
            "Id": "38fcecd8416a14796ba17134f1e882072b0fa84e",
            "Name": "author",
            "User": {
              "DataSourceId": "git",
              "Id": "582d5784c59885c497f615bd692bc031c42a18cc",
              "Name": "Łukasz Gryglicki"
            },
            "Weight": 1,
            "When": "2021-03-25T09:07:35.000Z",
            "WhenInTz": "2021-03-25T09:07:35.000Z"
          },
          {
            "Id": "cbb5ed2d58778e8a040465bdf6e6e6bf28275b64",
            "Name": "committer",
            "User": {
              "DataSourceId": "git",
              "Id": "582d5784c59885c497f615bd692bc031c42a18cc",
              "Name": "Łukasz Gryglicki"
            },
            "Weight": 1,
            "When": "2021-03-25T09:07:51.000Z",
            "WhenInTz": "2021-03-25T09:07:51.000Z"
          },
          {
            "Id": "1007c8826f8f3516feb01f3f743eccd7bc90bbab",
            "Name": "signer",
            "User": {
              "DataSourceId": "git",
              "Id": "582d5784c59885c497f615bd692bc031c42a18cc",
              "Name": "Łukasz Gryglicki"
            },
            "Weight": 1,
            "When": "2021-03-25T09:07:35.000Z",
            "WhenInTz": "2021-03-25T09:07:35.000Z"
          },
          {
            "Id": "8486b35297dc9156f37989a41d34f8ba22c250cd",
            "Name": "co_author",
            "User": {
              "DataSourceId": "git",
              "Id": "511a6708b8b02de7e41899173f7b4f62ccec1bc2",
              "Name": "Unicron"
            },
            "Weight": 1,
            "When": "2021-03-25T09:07:35.000Z",
            "WhenInTz": "2021-03-25T09:07:35.000Z"
          }
        ],
        "SHA": "0feccac2c7ae7dfb272a23d76c4cd471ab42b57d",
        "Stats": {
          "LinesAdded": 1,
          "LinesChanged": 1
        },
        "Title": "test PP"
      }
    },
    {
      "Commit": {
        "CommitURL": "https://github.com/lukaszgryglicki/trailers-test/commit/d465d4c2c75e6cb36fb0596aede5364713586cf2",
        "DataSourceId": "git",
        "Files": [
          {
            "Action": "A",
            "Added": 1,
            "Changed": 1,
            "Name": "file.txt"
          }
        ],
        "HashShort": "d465d4",
        "Id": "bb23406617171b8d3043b64dee9b9292574556b2",
        "IsDoc": true,
        "Message": "Non doc commit\n\nSigned-off-by: Łukasz Gryglicki <lukaszgryglicki@o2.pl>",
        "Parents": [
          "65690551576eea1052947433620a7a9bc3a899b3"
        ],
        "RepositoryShortURL": "trailers-test",
        "RepositoryType": "github",
        "RepositoryURL": "https://github.com/lukaszgryglicki/trailers-test",
        "Roles": [
          {
            "Id": "afe984e9ca9e70e516fb005367683a0b79b58cdf",
            "Name": "author",
            "User": {
              "DataSourceId": "git",
              "Id": "582d5784c59885c497f615bd692bc031c42a18cc",
              "Name": "Łukasz Gryglicki"
            },
            "Weight": 1,
            "When": "2021-03-25T08:14:36.000Z",
            "WhenInTz": "2021-03-25T08:14:36.000Z"
          },
          {
            "Id": "40e2b03b727349f132c9e7ade42c63ed019d9263",
            "Name": "committer",
            "User": {
              "DataSourceId": "git",
              "Id": "582d5784c59885c497f615bd692bc031c42a18cc",
              "Name": "Łukasz Gryglicki"
            },
            "Weight": 1,
            "When": "2021-03-25T08:14:36.000Z",
            "WhenInTz": "2021-03-25T08:14:36.000Z"
          },
          {
            "Id": "39cd6c58800d6e9279f480c4424de9d3bf78ab5a",
            "Name": "signer",
            "User": {
              "DataSourceId": "git",
              "Id": "582d5784c59885c497f615bd692bc031c42a18cc",
              "Name": "Łukasz Gryglicki"
            },
            "Weight": 1,
            "When": "2021-03-25T08:14:36.000Z",
            "WhenInTz": "2021-03-25T08:14:36.000Z"
          }
        ],
        "SHA": "d465d4c2c75e6cb36fb0596aede5364713586cf2",
        "Stats": {
          "LinesAdded": 1,
          "LinesChanged": 1
        },
        "Title": "Non doc commit"
      }
    },
    {
      "Commit": {
        "CommitURL": "https://github.com/lukaszgryglicki/trailers-test/commit/0012369253cf55c4c026229f629444abec9cf4e5",
        "DataSourceId": "git",
        "Files": [
          {
            "Action": "M",
            "Added": 1,
            "Changed": 1,
            "Name": "file.txt"
          }
        ],
        "HashShort": "001236",
        "Id": "0bc1414955349983dc12631de45dd101aef02929",
        "IsDoc": true,
        "Message": "test PP\n\nSigned-off-by: Łukasz Gryglicki <lukaszgryglicki@o2.pl>",
        "Parents": [
          "d465d4c2c75e6cb36fb0596aede5364713586cf2"
        ],
        "RepositoryShortURL": "trailers-test",
        "RepositoryType": "github",
        "RepositoryURL": "https://github.com/lukaszgryglicki/trailers-test",
        "Roles": [
          {
            "Id": "be3e9c7285f8dc18c83a23866ef6773d06331365",
            "Name": "author",
            "User": {
              "DataSourceId": "git",
              "Id": "582d5784c59885c497f615bd692bc031c42a18cc",
              "Name": "Łukasz Gryglicki"
            },
            "Weight": 1,
            "When": "2021-03-25T09:07:35.000Z",
            "WhenInTz": "2021-03-25T09:07:35.000Z"
          },
          {
            "Id": "517d4c9b662e283b1f9ab9c64376dc01e3e38594",
            "Name": "committer",
            "User": {
              "DataSourceId": "git",
              "Id": "582d5784c59885c497f615bd692bc031c42a18cc",
              "Name": "Łukasz Gryglicki"
            },
            "Weight": 1,
            "When": "2021-03-25T09:07:35.000Z",
            "WhenInTz": "2021-03-25T09:07:35.000Z"
          },
          {
            "Id": "41f5ec140ca44e931f3cdbb7d83f2dcd7709a524",
            "Name": "signer",
            "User": {
              "DataSourceId": "git",
              "Id": "582d5784c59885c497f615bd692bc031c42a18cc",
              "Name": "Łukasz Gryglicki"
            },
            "Weight": 1,
            "When": "2021-03-25T09:07:35.000Z",
            "WhenInTz": "2021-03-25T09:07:35.000Z"
          }
        ],
        "SHA": "0012369253cf55c4c026229f629444abec9cf4e5",
        "Stats": {
          "LinesAdded": 1,
          "LinesChanged": 1
        },
        "Title": "test PP"
      }
    },
    {
      "Commit": {
        "CommitURL": "https://github.com/lukaszgryglicki/trailers-test/commit/65690551576eea1052947433620a7a9bc3a899b3",
        "DataSourceId": "git",
        "Files": [
          {
            "Action": "M",
            "Added": 5,
            "Changed": 5,
            "Name": "README.md"
          }
        ],
        "HashShort": "656905",
        "Id": "31a6623f225d615257942c412e0f1a7221089a51",
        "IsDoc": true,
        "Message": "Various trailers\n\nSigned-off-by: Łukasz Gryglicki <lukaszgryglicki@o2.pl>\nCc: Pavel Machek <pavel@ucw.cz>\nAcked-by: Adam Belay <adam.belay@novell.com>\nSigned-Off-By: Adam Belay <abelay@novell.com>\nThanks-to: Dwaine Garden <DwaineGarden@rogers.com>\nACKed-by: Andi Kleen <ak@suse.de>\nAOLed-by: David Woodhouse <dwmw2@infradead.org>\nAcked-and-tested-by: Rob Landley <rob@landley.net>\nAll-the-fault-of: David Woodhouse <dwmw2@infradead.org>\nAnd: Tilman Schmidt <tilman@imap.cc>\nApology-from: Hugh Dickins <hugh@veritas.com>\nApproved-by: Rogier Wolff <R.E.Wolff@BitWizard.nl>\nBased-on-original-patch-by: Luming Yu <luming.yu@intel.com>\nBisected-and-requested-by: Heikki Orsila <shdl@zakalwe.fi>\nBisected-by: Arjan van de Ven <arjan@linux.intel.com>\nBitten-by-and-tested-by: Ingo Molnar <mingo@elte.hu>\nBug-found-by: Matthew Wilcox <matthew@wil.cx>\nBuild-fixes-from: Andrew Morton <akpm@osdl.org>",
        "Parents": [
          "0e5e87e7fcda7b6f133fb969d88ae8d0c68e3533"
        ],
        "RepositoryShortURL": "trailers-test",
        "RepositoryType": "github",
        "RepositoryURL": "https://github.com/lukaszgryglicki/trailers-test",
        "Roles": [
          {
            "Id": "1ba4b0b0bd32e45a03d0e515ff12d29befc5b5db",
            "Name": "author",
            "User": {
              "DataSourceId": "git",
              "Id": "582d5784c59885c497f615bd692bc031c42a18cc",
              "Name": "Łukasz Gryglicki"
            },
            "Weight": 1,
            "When": "2021-03-25T07:52:54.000Z",
            "WhenInTz": "2021-03-25T07:52:54.000Z"
          },
          {
            "Id": "6828dcc9d2e2c71d6803c9a08ed6af4b6a5d95f2",
            "Name": "committer",
            "User": {
              "DataSourceId": "git",
              "Id": "582d5784c59885c497f615bd692bc031c42a18cc",
              "Name": "Łukasz Gryglicki"
            },
            "Weight": 1,
            "When": "2021-03-25T07:52:58.000Z",
            "WhenInTz": "2021-03-25T07:52:58.000Z"
          },
          {
            "Id": "97274ce3cee02583a624d5631cb62483ac969d05",
            "Name": "influencer",
            "User": {
              "DataSourceId": "git",
              "Id": "199415491d2ca2090d3c56247785b76dd9eed20b",
              "Name": "Dwaine Garden"
            },
            "Weight": 1,
            "When": "2021-03-25T07:52:54.000Z",
            "WhenInTz": "2021-03-25T07:52:54.000Z"
          },
          {
            "Id": "de760da5a6e91717bda40a4d6ec41b28b96a3d46",
            "Name": "informer",
            "User": {
              "DataSourceId": "git",
              "Id": "199415491d2ca2090d3c56247785b76dd9eed20b",
              "Name": "Dwaine Garden"
            },
            "Weight": 1,
            "When": "2021-03-25T07:52:54.000Z",
            "WhenInTz": "2021-03-25T07:52:54.000Z"
          },
          {
            "Id": "1104b9672792a7ebf9eab10adf463a77bfaa500b",
            "Name": "reviewer",
            "User": {
              "DataSourceId": "git",
              "Id": "394de00b8d7dd753c945b3fc3f8a505c72fbdb52",
              "Name": "Adam Belay"
            },
            "Weight": 1,
            "When": "2021-03-25T07:52:54.000Z",
            "WhenInTz": "2021-03-25T07:52:54.000Z"
          },
          {
            "Id": "c941d30473d79d7d12a9221569958872dec4ec57",
            "Name": "reviewer",
            "User": {
              "DataSourceId": "git",
              "Id": "594c4fb655c9160e6d59ea19962cbebd948c9619",
              "Name": "Arjan van de Ven"
            },
            "Weight": 1,
            "When": "2021-03-25T07:52:54.000Z",
            "WhenInTz": "2021-03-25T07:52:54.000Z"
          },
          {
            "Id": "c9d6f3e42cc14a82020155c74504c93c502123d4",
            "Name": "reviewer",
            "User": {
              "DataSourceId": "git",
              "Id": "3343be6389ff917bcb4c5713168f264143c6ac4a",
              "Name": "David Woodhouse"
            },
            "Weight": 1,
            "When": "2021-03-25T07:52:54.000Z",
            "WhenInTz": "2021-03-25T07:52:54.000Z"
          },
          {
            "Id": "98f0ebcad6a7f90c8e6fe52181b9f0ccc9cf8ed8",
            "Name": "informer",
            "User": {
              "DataSourceId": "git",
              "Id": "3343be6389ff917bcb4c5713168f264143c6ac4a",
              "Name": "David Woodhouse"
            },
            "Weight": 1,
            "When": "2021-03-25T07:52:54.000Z",
            "WhenInTz": "2021-03-25T07:52:54.000Z"
          },
          {
            "Id": "e0855d6f738eec76853fb8bcafaaf185572cb49b",
            "Name": "reviewer",
            "User": {
              "DataSourceId": "git",
              "Id": "90177a03a93e01890ff6effb6bd1e12540a47ae5",
              "Name": "Rob Landley"
            },
            "Weight": 1,
            "When": "2021-03-25T07:52:54.000Z",
            "WhenInTz": "2021-03-25T07:52:54.000Z"
          },
          {
            "Id": "e5a5b769ba1e24afbf4b07e5a632420c4bba2f50",
            "Name": "tester",
            "User": {
              "DataSourceId": "git",
              "Id": "90177a03a93e01890ff6effb6bd1e12540a47ae5",
              "Name": "Rob Landley"
            },
            "Weight": 1,
            "When": "2021-03-25T07:52:54.000Z",
            "WhenInTz": "2021-03-25T07:52:54.000Z"
          },
          {
            "Id": "8e2a7b1a48050fe2e9795aceea56760bc620b7da",
            "Name": "reviewer",
            "User": {
              "DataSourceId": "git",
              "Id": "6cb1cf2e9beccd0f19bab2355cd9273c708c7099",
              "Name": "Ingo Molnar"
            },
            "Weight": 1,
            "When": "2021-03-25T07:52:54.000Z",
            "WhenInTz": "2021-03-25T07:52:54.000Z"
          },
          {
            "Id": "516d1e972d39b39f51f3a41f36a368102107d20b",
            "Name": "tester",
            "User": {
              "DataSourceId": "git",
              "Id": "6cb1cf2e9beccd0f19bab2355cd9273c708c7099",
              "Name": "Ingo Molnar"
            },
            "Weight": 1,
            "When": "2021-03-25T07:52:54.000Z",
            "WhenInTz": "2021-03-25T07:52:54.000Z"
          },
          {
            "Id": "c29c55a4937e4bb37743b8232a3d9019d47664e6",
            "Name": "signer",
            "User": {
              "DataSourceId": "git",
              "Id": "4120a9f78bf9cb6c3e33b602b95273b7a164ba08",
              "Name": "Adam Belay"
            },
            "Weight": 1,
            "When": "2021-03-25T07:52:54.000Z",
            "WhenInTz": "2021-03-25T07:52:54.000Z"
          },
          {
            "Id": "a4e8f0bd5b9374c466c4444d5f5f7ce6dbe58c65",
            "Name": "co_author",
            "User": {
              "DataSourceId": "git",
              "Id": "4120a9f78bf9cb6c3e33b602b95273b7a164ba08",
              "Name": "Adam Belay"
            },
            "Weight": 1,
            "When": "2021-03-25T07:52:54.000Z",
            "WhenInTz": "2021-03-25T07:52:54.000Z"
          },
          {
            "Id": "0f841cfa7a0d38af7e69ded42844c2a59aa64774",
            "Name": "approver",
            "User": {
              "DataSourceId": "git",
              "Id": "66c094ddea03b305c426798dc4264243dfd60e67",
              "Name": "Rogier Wolff"
            },
            "Weight": 1,
            "When": "2021-03-25T07:52:54.000Z",
            "WhenInTz": "2021-03-25T07:52:54.000Z"
          },
          {
            "Id": "269306ca64a0fdb34e888d50cd274992d36c7765",
            "Name": "informer",
            "User": {
              "DataSourceId": "git",
              "Id": "186a5830b3d89ff6809634ff884d5b7405052b37",
              "Name": "Pavel Machek"
            },
            "Weight": 1,
            "When": "2021-03-25T07:52:54.000Z",
            "WhenInTz": "2021-03-25T07:52:54.000Z"
          },
          {
            "Id": "0113fcc2cca3c21bcd44b39bfbfd75afe5a571c9",
            "Name": "reviewer",
            "User": {
              "DataSourceId": "git",
              "Id": "30c4ea68741d54ceb4e98470775a85c937a100cc",
              "Name": "Andi Kleen"
            },
            "Weight": 1,
            "When": "2021-03-25T07:52:54.000Z",
            "WhenInTz": "2021-03-25T07:52:54.000Z"
          },
          {
            "Id": "23f96ca3ea89c55557f04c0c90ed12d9cf3d5de0",
            "Name": "reporter",
            "User": {
              "DataSourceId": "git",
              "Id": "9256d61633ee91c3c1ff8d83ea1f5a0cb7b3e996",
              "Name": "Matthew Wilcox"
            },
            "Weight": 1,
            "When": "2021-03-25T07:52:54.000Z",
            "WhenInTz": "2021-03-25T07:52:54.000Z"
          },
          {
            "Id": "7cdc3a4f984d22e2b07afa02e1223b426233823a",
            "Name": "influencer",
            "User": {
              "DataSourceId": "git",
              "Id": "11c0cb6ccce70793af4050e03cc73b35991272cf",
              "Name": "Luming Yu"
            },
            "Weight": 1,
            "When": "2021-03-25T07:52:54.000Z",
            "WhenInTz": "2021-03-25T07:52:54.000Z"
          },
          {
            "Id": "df982692c0d28551cfb95d15795adf8f2f64cc35",
            "Name": "resolver",
            "User": {
              "DataSourceId": "git",
              "Id": "99f4cf07b6ecb7a038007a24d4c1cc518c14d34a",
              "Name": "Andrew Morton"
            },
            "Weight": 1,
            "When": "2021-03-25T07:52:54.000Z",
            "WhenInTz": "2021-03-25T07:52:54.000Z"
          },
          {
            "Id": "e7e1f085897bf19cca7d5428455867aef9a8df6d",
            "Name": "signer",
            "User": {
              "DataSourceId": "git",
              "Id": "582d5784c59885c497f615bd692bc031c42a18cc",
              "Name": "Łukasz Gryglicki"
            },
            "Weight": 1,
            "When": "2021-03-25T07:52:54.000Z",
            "WhenInTz": "2021-03-25T07:52:54.000Z"
          },
          {
            "Id": "fa99cbfd73cd5c9c3d6b0cc7c2e19f72be720ccd",
            "Name": "informer",
            "User": {
              "DataSourceId": "git",
              "Id": "06cc03ef357d5b73cb57e4d376382319cceb814a",
              "Name": "Hugh Dickins"
            },
            "Weight": 1,
            "When": "2021-03-25T07:52:54.000Z",
            "WhenInTz": "2021-03-25T07:52:54.000Z"
          }
        ],
        "SHA": "65690551576eea1052947433620a7a9bc3a899b3",
        "Stats": {
          "LinesAdded": 5,
          "LinesChanged": 5
        },
        "Title": "Various trailers"
      }
    },
    {
      "Commit": {
        "CommitURL": "https://github.com/lukaszgryglicki/trailers-test/commit/38165eecda3c01e17b47269e8a576103d8257efd",
        "DataSourceId": "git",
        "Files": [
          {
            "Action": "D",
            "Changed": 3,
            "Name": "file.txt",
            "Removed": 3
          }
        ],
        "HashShort": "38165e",
        "Id": "8de4ffa8337210fb3ad689b71d42608355882465",
        "IsDoc": true,
        "Message": "Delete file action\n\nSigned-off-by: Łukasz Gryglicki <lukaszgryglicki@o2.pl>",
        "Parents": [
          "878e8678dec2f2bbcbb85b02eccafa6095b68c1b"
        ],
        "RepositoryShortURL": "trailers-test",
        "RepositoryType": "github",
        "RepositoryURL": "https://github.com/lukaszgryglicki/trailers-test",
        "Roles": [
          {
            "Id": "a34fbcc2db9c51f9de06a959d9fb6ce2eee21f48",
            "Name": "author",
            "User": {
              "DataSourceId": "git",
              "Id": "582d5784c59885c497f615bd692bc031c42a18cc",
              "Name": "Łukasz Gryglicki"
            },
            "Weight": 1,
            "When": "2021-03-25T17:03:30.000Z",
            "WhenInTz": "2021-03-25T17:03:30.000Z"
          },
          {
            "Id": "5cce66ce672a6be2f664604a7e685403a16b9c20",
            "Name": "committer",
            "User": {
              "DataSourceId": "git",
              "Id": "582d5784c59885c497f615bd692bc031c42a18cc",
              "Name": "Łukasz Gryglicki"
            },
            "Weight": 1,
            "When": "2021-03-25T17:03:30.000Z",
            "WhenInTz": "2021-03-25T17:03:30.000Z"
          },
          {
            "Id": "968b82c33c2f9588472e2d64d44997e52f79b195",
            "Name": "signer",
            "User": {
              "DataSourceId": "git",
              "Id": "582d5784c59885c497f615bd692bc031c42a18cc",
              "Name": "Łukasz Gryglicki"
            },
            "Weight": 1,
            "When": "2021-03-25T17:03:30.000Z",
            "WhenInTz": "2021-03-25T17:03:30.000Z"
          }
        ],
        "SHA": "38165eecda3c01e17b47269e8a576103d8257efd",
        "Stats": {
          "LinesChanged": 3,
          "LinesRemoved": 3
        },
        "Title": "Delete file action"
      }
    },
    {
      "Commit": {
        "CommitURL": "https://github.com/lukaszgryglicki/trailers-test/commit/878e8678dec2f2bbcbb85b02eccafa6095b68c1b",
        "DataSourceId": "git",
        "Files": [
          {
            "Action": "M",
            "Added": 1,
            "Changed": 1,
            "Name": "file.txt"
          }
        ],
        "HashShort": "878e86",
        "Id": "34a25e33b2ad9d21d9b5ff1db0ed9e42ad44e11b",
        "IsDoc": true,
        "Message": "Ne wtrailers test\n\nSigned-off-by: Łukasz Gryglicki <lukaszgryglicki@o2.pl>\nsigned-off: new identity <new-identity@mail.uv>\nsigned-off-by: new identity <new-identity@mail.uv>\nco-authored: new identity <new-identity@mail.uv>\nco-authored-by: new identity <new-identity@mail.uv>\ncc: new identity <new-identity@mail.uv>\ncc-by: new identity <new-identity@mail.uv>\nack: new identity <new-identity@mail.uv>\nack-by: new identity <new-identity@mail.uv>\nacked: new identity <new-identity@mail.uv>\nacked-by: new identity <new-identity@mail.uv>\nacked-and-tested-by: new identity <new-identity@mail.uv>\ntested: new identity <new-identity@mail.uv>\ntested-by: new identity <new-identity@mail.uv>\napproved: new identity <new-identity@mail.uv>\napproved-by: new identity <new-identity@mail.uv>\nacked-and-reviewed-by: new identity <new-identity@mail.uv>\nacked-and-reviewed: new identity <new-identity@mail.uv>\nreviewed: new identity <new-identity@mail.uv>\nreviewed-by: new identity <new-identity@mail.uv>\nlooks-good-to: new identity <new-identity@mail.uv>\nanalyzed: new identity <new-identity@mail.uv>\nanalyzed-by: new identity <new-identity@mail.uv>\nreported: new identity <new-identity@mail.uv>\nreported-by: new identity <new-identity@mail.uv>\ncommitted: new identity <new-identity@mail.uv>\ncommitted-by: new identity <new-identity@mail.uv>",
        "Parents": [
          "ec71bbb88cb677f8c5fbaf6962ade34c1348ed73"
        ],
        "RepositoryShortURL": "trailers-test",
        "RepositoryType": "github",
        "RepositoryURL": "https://github.com/lukaszgryglicki/trailers-test",
        "Roles": [
          {
            "Id": "f78a710e03d99c315cab398654215a28d55a6148",
            "Name": "author",
            "User": {
              "DataSourceId": "git",
              "Id": "582d5784c59885c497f615bd692bc031c42a18cc",
              "Name": "Łukasz Gryglicki"
            },
            "Weight": 1,
            "When": "2021-03-25T10:19:04.000Z",
            "WhenInTz": "2021-03-25T10:19:04.000Z"
          },
          {
            "Id": "3b3734cebd69062fb7abbf34b6e80136b693070d",
            "Name": "committer",
            "User": {
              "DataSourceId": "git",
              "Id": "582d5784c59885c497f615bd692bc031c42a18cc",
              "Name": "Łukasz Gryglicki"
            },
            "Weight": 1,
            "When": "2021-03-25T10:19:10.000Z",
            "WhenInTz": "2021-03-25T10:19:10.000Z"
          },
          {
            "Id": "8fe0a68ee6ecb2e7561c9bcde963ef663e11254e",
            "Name": "reporter",
            "User": {
              "DataSourceId": "git",
              "Id": "3260cf5673f225ed6b579d22466a10cf1f560ef9",
              "Name": "new identity"
            },
            "Weight": 1,
            "When": "2021-03-25T10:19:04.000Z",
            "WhenInTz": "2021-03-25T10:19:04.000Z"
          },
          {
            "Id": "6f89436cd3b47cf891391a50703dc46818c867b9",
            "Name": "tester",
            "User": {
              "DataSourceId": "git",
              "Id": "3260cf5673f225ed6b579d22466a10cf1f560ef9",
              "Name": "new identity"
            },
            "Weight": 1,
            "When": "2021-03-25T10:19:04.000Z",
            "WhenInTz": "2021-03-25T10:19:04.000Z"
          },
          {
            "Id": "fce6580b53333825ad8497b5392b7374a0363d8b",
            "Name": "informer",
            "User": {
              "DataSourceId": "git",
              "Id": "3260cf5673f225ed6b579d22466a10cf1f560ef9",
              "Name": "new identity"
            },
            "Weight": 1,
            "When": "2021-03-25T10:19:04.000Z",
            "WhenInTz": "2021-03-25T10:19:04.000Z"
          },
          {
            "Id": "e74763f375dc8ba5e36eaf1a0158b7df28becdab",
            "Name": "signer",
            "User": {
              "DataSourceId": "git",
              "Id": "3260cf5673f225ed6b579d22466a10cf1f560ef9",
              "Name": "new identity"
            },
            "Weight": 1,
            "When": "2021-03-25T10:19:04.000Z",
            "WhenInTz": "2021-03-25T10:19:04.000Z"
          },
          {
            "Id": "ab7e6844aef07691f3d89bbfec7704729e278d96",
            "Name": "co_author",
            "User": {
              "DataSourceId": "git",
              "Id": "3260cf5673f225ed6b579d22466a10cf1f560ef9",
              "Name": "new identity"
            },
            "Weight": 1,
            "When": "2021-03-25T10:19:04.000Z",
            "WhenInTz": "2021-03-25T10:19:04.000Z"
          },
          {
            "Id": "004dc3b976003d60a26fd664baf1c140ad0a45c6",
            "Name": "approver",
            "User": {
              "DataSourceId": "git",
              "Id": "3260cf5673f225ed6b579d22466a10cf1f560ef9",
              "Name": "new identity"
            },
            "Weight": 1,
            "When": "2021-03-25T10:19:04.000Z",
            "WhenInTz": "2021-03-25T10:19:04.000Z"
          },
          {
            "Id": "d1fd75d3e9f0a268e5172e571f8561ae22edcd9b",
            "Name": "reviewer",
            "User": {
              "DataSourceId": "git",
              "Id": "3260cf5673f225ed6b579d22466a10cf1f560ef9",
              "Name": "new identity"
            },
            "Weight": 1,
            "When": "2021-03-25T10:19:04.000Z",
            "WhenInTz": "2021-03-25T10:19:04.000Z"
          },
          {
            "Id": "3495ac3d3ba53a244f68f454478f7765bde11d66",
            "Name": "signer",
            "User": {
              "DataSourceId": "git",
              "Id": "582d5784c59885c497f615bd692bc031c42a18cc",
              "Name": "Łukasz Gryglicki"
            },
            "Weight": 1,
            "When": "2021-03-25T10:19:04.000Z",
            "WhenInTz": "2021-03-25T10:19:04.000Z"
          }
        ],
        "SHA": "878e8678dec2f2bbcbb85b02eccafa6095b68c1b",
        "Stats": {
          "LinesAdded": 1,
          "LinesChanged": 1
        },
        "Title": "Ne wtrailers test"
      }
    },
    {
      "Commit": {
        "CommitURL": "https://github.com/lukaszgryglicki/trailers-test/commit/a5dc87fbb1357518a9a1ea19a533977fbc74c819",
        "DataSourceId": "git",
        "Files": [
          {
            "Action": "M",
            "Added": 1,
            "Changed": 2,
            "Name": "README.md",
            "Removed": 1
          }
        ],
        "HashShort": "a5dc87",
        "Id": "17b33a9c985a94ec3bc6a50e0872ece95ab70322",
        "IsDoc": true,
        "Message": "Missing dot\n\nSigned-off-by: Łukasz Gryglicki <lukaszgryglicki@o2.pl>\nAuthored-by: David Woodhouse <dwmw2@infradead.org> and Tilman Schmidt <tilman@imap.cc>\nAuthor: David Woodhouse <dwmw2@infradead.org> and Tilman Schmidt <tilman@imap.cc>\nCommit: David Woodhouse <dwmw2@infradead.org> and Tilman Schmidt <tilman@imap.cc>\nCo-authored-by: Andi Kleen <ak@suse.de>\nCo-authored-by: Andi2 Kleen2 <ak2@suse.de>",
        "Parents": [
          "38165eecda3c01e17b47269e8a576103d8257efd"
        ],
        "RepositoryShortURL": "trailers-test",
        "RepositoryType": "github",
        "RepositoryURL": "https://github.com/lukaszgryglicki/trailers-test",
        "Roles": [
          {
            "Id": "a46ba8484953e9d5b41907a1831d954d8ed26fbe",
            "Name": "author",
            "User": {
              "DataSourceId": "git",
              "Id": "582d5784c59885c497f615bd692bc031c42a18cc",
              "Name": "Łukasz Gryglicki"
            },
            "Weight": 1,
            "When": "2021-08-17T09:11:13.000Z",
            "WhenInTz": "2021-08-17T09:11:13.000Z"
          },
          {
            "Id": "00e0f4e97908ee76f64589e1c3cb7a037593f782",
            "Name": "committer",
            "User": {
              "DataSourceId": "git",
              "Id": "582d5784c59885c497f615bd692bc031c42a18cc",
              "Name": "Łukasz Gryglicki"
            },
            "Weight": 1,
            "When": "2021-08-17T09:11:17.000Z",
            "WhenInTz": "2021-08-17T09:11:17.000Z"
          },
          {
            "Id": "6cb9ccc0b6f7e47172f2bec75f30e8653e671637",
            "Name": "signer",
            "User": {
              "DataSourceId": "git",
              "Id": "582d5784c59885c497f615bd692bc031c42a18cc",
              "Name": "Łukasz Gryglicki"
            },
            "Weight": 1,
            "When": "2021-08-17T09:11:13.000Z",
            "WhenInTz": "2021-08-17T09:11:13.000Z"
          },
          {
            "Id": "ef125538e11df067bea212375974f171ceb8511c",
            "Name": "co_author",
            "User": {
              "DataSourceId": "git",
              "Id": "30c4ea68741d54ceb4e98470775a85c937a100cc",
              "Name": "Andi Kleen"
            },
            "Weight": 1,
            "When": "2021-08-17T09:11:13.000Z",
            "WhenInTz": "2021-08-17T09:11:13.000Z"
          },
          {
            "Id": "96ea6439fd775fac377ddc49aff871733f80ac52",
            "Name": "co_author",
            "User": {
              "DataSourceId": "git",
              "Id": "f27c7455bccdca659f055688fb32d644e7d6e688",
              "Name": "Andi2 Kleen2"
            },
            "Weight": 1,
            "When": "2021-08-17T09:11:13.000Z",
            "WhenInTz": "2021-08-17T09:11:13.000Z"
          }
        ],
        "SHA": "a5dc87fbb1357518a9a1ea19a533977fbc74c819",
        "Stats": {
          "LinesAdded": 1,
          "LinesChanged": 2,
          "LinesRemoved": 1
        },
        "Title": "Missing dot"
      }
    }
  ],
  "MetaData": {
    "BackendName": "git",
    "BackendVersion": "0.1.1",
    "Tags": null
  },
  "RepositoryStats": {
    "CalculatedAt": "2021-08-17T12:54:22.109Z",
    "ProgrammingLanguagesStats": [
      {
        "Blank": 3,
        "CalculatedAt": "2021-08-17T12:54:22.109Z",
        "Code": 4,
        "Files": 1,
        "Language": "Markdown"
      }
    ],
    "RepositoryURL": "https://github.com/lukaszgryglicki/trailers-test",
    "TotalLinesOfCode": 4
  }
}
```
