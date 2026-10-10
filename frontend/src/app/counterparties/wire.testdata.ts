// Trimmed Go IPC specimens; test-only, never a browser fallback.
export const surfaceWire = {
  "value": {
    "summary": "3810 trade sides of valued trades, 2017–2026. On the day is " +
      "expected wins gained at the league's own prices; since is wins " +
      "actually delivered, for trades with two or more seasons played.",
    "asOf": "Standings as of the archive cutoff 2026-10-06, not today's live standings.",
    "headline": "Contenders buy for now in 24.9% of their trades and rebuilding " +
      "teams build for later in 25.3%; most trades on both sides are " +
      "like for like. No tier wins its trades on the day beyond chance: " +
      "the edge is in the teams, not the standings.",
    "note": "Each trade has two sides, one per team. A team's tier is its " +
      "rank that day: live once three games are final, else last " +
      "season's finish, as the archive recorded it.",
    "picks": [
      {
        "id": "contender",
        "label": "Contenders",
        "hint": "Ranked 1 to 10 on the day of the trade",
        "count": 157.7
      },
      {
        "id": "hunt",
        "label": "In the hunt",
        "hint": "Ranked 11 to 22 on the day of the trade",
        "count": 108.4
      },
      {
        "id": "rebuild",
        "label": "Rebuilding",
        "hint": "Ranked 23 to 32 on the day of the trade",
        "count": 114.9
      }
    ],
    "teams": [
      {
        "fid": "0025",
        "name": "Arizona Cardinals"
      },
      {
        "fid": "0017",
        "name": "Atlanta Falcons"
      }
    ],
    "cards": [
      {
        "id": "contender",
        "label": "Contenders",
        "hint": "Ranked 1 to 10 on the day of the trade",
        "thin": false,
        "perSeason": 157.7,
        "perSeasonText": "157.7 trade sides a season",
        "posture": [
          {
            "id": "now",
            "label": "Bought for now",
            "p": 0.24920735573874445,
            "tone": "even",
            "title": "Bought for now in 24.9% of these trades, within chance of the league's share"
          },
          {
            "id": "later",
            "label": "Built for later",
            "p": 0.21623335447051364,
            "tone": "down",
            "title": "Built for later in 21.6% of these trades, below the league's " +
              "share beyond chance"
          },
          {
            "id": "swap",
            "label": "Like for like",
            "p": 0.534559289790742,
            "tone": "even",
            "title": "Like for like in 53.5% of these trades, within chance of the league's share"
          }
        ],
        "rows": [
          {
            "label": "On the day",
            "text": "-0.01 expected wins a trade, within chance of zero",
            "tone": "even",
            "range": {
              "n": 1577,
              "mean": -0.007635383639822449,
              "lo": -0.02792156581135804,
              "hi": 0.012650798531713138,
              "total": -12.041000000000002,
              "tone": "even"
            }
          },
          {
            "label": "Since (2+ seasons played)",
            "text": "+0.03 wins a trade, within chance of zero",
            "tone": "even",
            "range": {
              "n": 659,
              "mean": 0.028216995447647948,
              "lo": -0.03779985194036708,
              "hi": 0.09423384283566297,
              "total": 18.595,
              "tone": "even"
            }
          },
          {
            "label": "Picks in minus out",
            "text": "-0.05 a trade",
            "tone": "even",
            "range": {
              "n": 1577,
              "mean": -0.054533925174381735,
              "lo": -0.1340375884023219,
              "hi": 0.024969738053558434,
              "total": -86,
              "tone": "even"
            }
          },
          {
            "label": "Age in minus out",
            "text": "+0.1 years",
            "tone": "even",
            "range": {
              "n": 989,
              "mean": 0.10929582550917238,
              "lo": -0.1008217941727946,
              "hi": 0.31941344519113934,
              "total": 108.0935714285715,
              "tone": "even"
            }
          },
          {
            "label": "Trades with",
            "text": "contenders 33.0% · in the hunt 32.3% · rebuilding 34.7%",
            "tone": ""
          }
        ]
      },
      {
        "id": "hunt",
        "label": "In the hunt",
        "hint": "Ranked 11 to 22 on the day of the trade",
        "thin": false,
        "perSeason": 108.4,
        "perSeasonText": "108.4 trade sides a season",
        "posture": [
          {
            "id": "now",
            "label": "Bought for now",
            "p": 0.24723247232472326,
            "tone": "even",
            "title": "Bought for now in 24.7% of these trades, within chance of the league's share"
          },
          {
            "id": "later",
            "label": "Built for later",
            "p": 0.2730627306273063,
            "tone": "up",
            "title": "Built for later in 27.3% of these trades, above the league's " +
              "share beyond chance"
          },
          {
            "id": "swap",
            "label": "Like for like",
            "p": 0.4797047970479705,
            "tone": "down",
            "title": "Like for like in 48.0% of these trades, below the league's share " +
              "beyond chance"
          }
        ],
        "rows": [
          {
            "label": "On the day",
            "text": "-0.00 expected wins a trade, within chance of zero",
            "tone": "even",
            "range": {
              "n": 1084,
              "mean": -0.0031808118081180768,
              "lo": -0.028678346443965577,
              "hi": 0.02231672282772942,
              "total": -3.447999999999995,
              "tone": "even"
            }
          },
          {
            "label": "Since (2+ seasons played)",
            "text": "-0.08 wins a trade, within chance of zero",
            "tone": "even",
            "range": {
              "n": 436,
              "mean": -0.07785779816513765,
              "lo": -0.16522170638594968,
              "hi": 0.009506110055674408,
              "total": -33.94600000000001,
              "tone": "even"
            }
          },
          {
            "label": "Picks in minus out",
            "text": "+0.06 a trade",
            "tone": "even",
            "range": {
              "n": 1084,
              "mean": 0.061808118081180814,
              "lo": -0.03882655165103153,
              "hi": 0.16244278781339316,
              "total": 67,
              "tone": "even"
            }
          },
          {
            "label": "Age in minus out",
            "text": "-0.1 years",
            "tone": "even",
            "range": {
              "n": 633,
              "mean": -0.08986000834342059,
              "lo": -0.3205308773805622,
              "hi": 0.14081086069372098,
              "total": -56.881385281385235,
              "tone": "even"
            }
          },
          {
            "label": "Trades with",
            "text": "contenders 47.0% · in the hunt 28.0% · rebuilding 24.9%",
            "tone": ""
          }
        ]
      },
      {
        "id": "rebuild",
        "label": "Rebuilding",
        "hint": "Ranked 23 to 32 on the day of the trade",
        "thin": false,
        "perSeason": 114.9,
        "perSeasonText": "114.9 trade sides a season",
        "posture": [
          {
            "id": "now",
            "label": "Bought for now",
            "p": 0.23237597911227154,
            "tone": "even",
            "title": "Bought for now in 23.2% of these trades, within chance of the league's share"
          },
          {
            "id": "later",
            "label": "Built for later",
            "p": 0.25326370757180156,
            "tone": "even",
            "title": "Built for later in 25.3% of these trades, within chance of the " +
              "league's share"
          },
          {
            "id": "swap",
            "label": "Like for like",
            "p": 0.5143603133159269,
            "tone": "even",
            "title": "Like for like in 51.4% of these trades, within chance of the league's share"
          }
        ],
        "rows": [
          {
            "label": "On the day",
            "text": "+0.01 expected wins a trade, within chance of zero",
            "tone": "even",
            "range": {
              "n": 1149,
              "mean": 0.013480417754569181,
              "lo": -0.01209138970123458,
              "hi": 0.039052225210372944,
              "total": 15.488999999999988,
              "tone": "even"
            }
          },
          {
            "label": "Since (2+ seasons played)",
            "text": "+0.04 wins a trade, within chance of zero",
            "tone": "even",
            "range": {
              "n": 427,
              "mean": 0.03595081967213111,
              "lo": -0.05151763605688131,
              "hi": 0.12341927540114353,
              "total": 15.350999999999983,
              "tone": "even"
            }
          },
          {
            "label": "Picks in minus out",
            "text": "+0.02 a trade",
            "tone": "even",
            "range": {
              "n": 1149,
              "mean": 0.016536118363794605,
              "lo": -0.08775615852063527,
              "hi": 0.12082839524822449,
              "total": 19,
              "tone": "even"
            }
          },
          {
            "label": "Age in minus out",
            "text": "-0.1 years",
            "tone": "even",
            "range": {
              "n": 560,
              "mean": -0.09145033240568977,
              "lo": -0.3801120059725381,
              "hi": 0.19721134116115852,
              "total": -51.21218614718627,
              "tone": "even"
            }
          },
          {
            "label": "Trades with",
            "text": "contenders 47.6% · in the hunt 23.5% · rebuilding 28.9%",
            "tone": ""
          }
        ]
      }
    ],
    "trendSay": [
      "Contenders bought for now in 21.9% of their trades in 2017–21 " +
        "and 27.0% in 2022–26.",
      "Contenders built for later in 18.4% of their trades in 2017–21 " +
        "and 23.8% in 2022–26.",
      "Every other habit held within chance between the two halves."
    ],
    "trends": {
      "seasons": [
        2017,
        2018,
        2019,
        2020,
        2021,
        2022,
        2023,
        2024,
        2025,
        2026
      ],
      "tiers": [
        {
          "tier": "contender",
          "rows": [
            {
              "season": 2017,
              "n": 82,
      "showShares": true,
              "now": {
                "k": 20,
                "n": 82,
                "p": 0.24390243902439024,
                "lo": 0.15095313647579134,
                "hi": 0.33685174157298914,
                "tone": "even"
              },
              "later": {
                "k": 15,
                "n": 82,
                "p": 0.18292682926829268,
                "lo": 0.09924747433332365,
                "hi": 0.2666061842032617,
                "tone": "even"
              },
              "net": 0.01224390243902438
            },
            {
              "season": 2026,
              "n": 91,
      "showShares": true,
              "now": {
                "k": 28,
                "n": 91,
                "p": 0.3076923076923077,
                "lo": 0.2128628486165403,
                "hi": 0.4025217667680751,
                "tone": "even"
              },
              "later": {
                "k": 24,
                "n": 91,
                "p": 0.26373626373626374,
                "lo": 0.17319699028869684,
                "hi": 0.35427553718383065,
                "tone": "even"
              },
              "net": -0.002879120879120877
            }
          ],
          "shift": {
            "later": {
              "early": 0.184375,
              "late": 0.23799359658484526,
              "from": [
                2017,
                2021
              ],
              "to": [
                2022,
                2026
              ],
              "clear": true
            },
            "now": {
              "early": 0.21875,
              "late": 0.2700106723585913,
              "from": [
                2017,
                2021
              ],
              "to": [
                2022,
                2026
              ],
              "clear": true
            }
          }
        },
        {
          "tier": "hunt",
          "rows": [
            {
              "season": 2017,
              "n": 66,
      "showShares": true,
              "now": {
                "k": 14,
                "n": 66,
                "p": 0.21212121212121213,
                "lo": 0.11349193326983396,
                "hi": 0.3107504909725903,
                "tone": "even"
              },
              "later": {
                "k": 13,
                "n": 66,
                "p": 0.19696969696969696,
                "lo": 0.1010186365015963,
                "hi": 0.2929207574377976,
                "tone": "even"
              },
              "net": -0.015545454545454532
            },
            {
              "season": 2026,
              "n": 53,
      "showShares": true,
              "now": {
                "k": 14,
                "n": 53,
                "p": 0.2641509433962264,
                "lo": 0.14545425886013663,
                "hi": 0.3828476279323162,
                "tone": "even"
              },
              "later": {
                "k": 20,
                "n": 53,
                "p": 0.37735849056603776,
                "lo": 0.24685737183121778,
                "hi": 0.5078596093008577,
                "tone": "even"
              },
              "net": -0.02032075471698113
            }
          ],
          "shift": {
            "later": {
              "early": 0.24731182795698925,
              "late": 0.2924071082390953,
              "from": [
                2017,
                2021
              ],
              "to": [
                2022,
                2026
              ],
              "clear": false
            },
            "now": {
              "early": 0.23010752688172043,
              "late": 0.26009693053311794,
              "from": [
                2017,
                2021
              ],
              "to": [
                2022,
                2026
              ],
              "clear": false
            }
          }
        },
        {
          "tier": "rebuild",
          "rows": [
            {
              "season": 2017,
              "n": 72,
      "showShares": true,
              "now": {
                "k": 13,
                "n": 72,
                "p": 0.18055555555555555,
                "lo": 0.09170605135641288,
                "hi": 0.2694050597546982,
                "tone": "even"
              },
              "later": {
                "k": 19,
                "n": 72,
                "p": 0.2638888888888889,
                "lo": 0.1620831964867105,
                "hi": 0.3656945812910673,
                "tone": "even"
              },
              "net": 0.0003055555555555481
            },
            {
              "season": 2026,
              "n": 70,
      "showShares": true,
              "now": {
                "k": 22,
                "n": 70,
                "p": 0.3142857142857143,
                "lo": 0.20553278329072916,
                "hi": 0.4230386452806994,
                "tone": "even"
              },
              "later": {
                "k": 20,
                "n": 70,
                "p": 0.2857142857142857,
                "lo": 0.17988423327170208,
                "hi": 0.3915443381568693,
                "tone": "even"
              },
              "net": 0.019128571428571432
            }
          ],
          "shift": {
            "later": {
              "early": 0.2346723044397463,
              "late": 0.26627218934911245,
              "from": [
                2017,
                2021
              ],
              "to": [
                2022,
                2026
              ],
              "clear": false
            },
            "now": {
              "early": 0.20507399577167018,
              "late": 0.2514792899408284,
              "from": [
                2017,
                2021
              ],
              "to": [
                2022,
                2026
              ],
              "clear": false
            }
          }
        }
      ]
    },
    "pairs": [
      {
        "other": "contender",
        "n": 520,
        "now": {
          "n": 520,
          "mean": 0,
          "lo": -0.034459004860177805,
          "hi": 0.034459004860177805,
          "total": 0,
          "tone": "even"
        },
        "real": {
          "n": 234,
          "mean": 0,
          "lo": -0.10758823794599204,
          "hi": 0.10758823794599204,
          "total": 0,
          "tone": "even"
        },
        "trades": "520 trades",
        "title": "Contenders trading with contenders: +0.00 expected wins a trade, " +
          "within chance of zero on the day; +0.00 wins a trade, within " +
          "chance of zero since."
      },
      {
        "other": "hunt",
        "n": 510,
        "now": {
          "n": 510,
          "mean": -0.006070588235294121,
          "lo": -0.04162863085038977,
          "hi": 0.029487454379801534,
          "total": -3.096000000000002,
          "tone": "even"
        },
        "real": {
          "n": 218,
          "mean": 0.106545871559633,
          "lo": -0.013781383696772731,
          "hi": 0.22687312681603874,
          "total": 23.226999999999997,
          "tone": "even"
        },
        "trades": "510 trades",
        "title": "Contenders trading with in the hunt: -0.01 expected wins a " +
          "trade, within chance of zero on the day; +0.11 wins a trade, " +
          "within chance of zero since."
      },
      {
        "other": "rebuild",
        "n": 547,
        "now": {
          "n": 547,
          "mean": -0.01635283363802558,
          "lo": -0.05173211353108326,
          "hi": 0.019026446255032095,
          "total": -8.944999999999993,
          "tone": "even"
        },
        "real": {
          "n": 207,
          "mean": -0.02237681159420293,
          "lo": -0.13757651970755352,
          "hi": 0.09282289651914767,
          "total": -4.632000000000007,
          "tone": "even"
        },
        "trades": "547 trades",
        "title": "Contenders trading with rebuilding: -0.02 expected wins a trade, " +
          "within chance of zero on the day; -0.02 wins a trade, within " +
          "chance of zero since."
      },
      {
        "other": "contender",
        "n": 510,
        "now": {
          "n": 510,
          "mean": 0.006070588235294121,
          "lo": -0.029487454379801534,
          "hi": 0.04162863085038977,
          "total": 3.096000000000002,
          "tone": "even"
        },
        "real": {
          "n": 218,
          "mean": -0.106545871559633,
          "lo": -0.22687312681603874,
          "hi": 0.013781383696772731,
          "total": -23.226999999999997,
          "tone": "even"
        },
        "trades": "510 trades",
        "title": "In the hunt trading with contenders: +0.01 expected wins a " +
          "trade, within chance of zero on the day; -0.11 wins a trade, " +
          "within chance of zero since."
      },
      {
        "other": "hunt",
        "n": 304,
        "now": {
          "n": 304,
          "mean": 0,
          "lo": -0.04741202057401206,
          "hi": 0.04741202057401206,
          "total": 0,
          "tone": "even"
        },
        "real": {
          "n": 106,
          "mean": 0,
          "lo": -0.15265409918665984,
          "hi": 0.15265409918665984,
          "total": 0,
          "tone": "even"
        },
        "trades": "304 trades",
        "title": "In the hunt trading with in the hunt: +0.00 expected wins a " +
          "trade, within chance of zero on the day; +0.00 wins a trade, " +
          "within chance of zero since."
      },
      {
        "other": "rebuild",
        "n": 270,
        "now": {
          "n": 270,
          "mean": -0.02423703703703705,
          "lo": -0.08018084757307245,
          "hi": 0.03170677349899834,
          "total": -6.544000000000003,
          "tone": "even"
        },
        "real": {
          "n": 112,
          "mean": -0.09570535714285718,
          "lo": -0.29624834237562425,
          "hi": 0.10483762808990989,
          "total": -10.719000000000003,
          "tone": "even"
        },
        "trades": "270 trades",
        "title": "In the hunt trading with rebuilding: -0.02 expected wins a " +
          "trade, within chance of zero on the day; -0.10 wins a trade, " +
          "within chance of zero since."
      },
      {
        "other": "contender",
        "n": 547,
        "now": {
          "n": 547,
          "mean": 0.01635283363802558,
          "lo": -0.019026446255032095,
          "hi": 0.05173211353108326,
          "total": 8.944999999999993,
          "tone": "even"
        },
        "real": {
          "n": 207,
          "mean": 0.02237681159420293,
          "lo": -0.09282289651914767,
          "hi": 0.13757651970755352,
          "total": 4.632000000000007,
          "tone": "even"
        },
        "trades": "547 trades",
        "title": "Rebuilding trading with contenders: +0.02 expected wins a trade, " +
          "within chance of zero on the day; +0.02 wins a trade, within " +
          "chance of zero since."
      },
      {
        "other": "hunt",
        "n": 270,
        "now": {
          "n": 270,
          "mean": 0.02423703703703705,
          "lo": -0.03170677349899834,
          "hi": 0.08018084757307245,
          "total": 6.544000000000003,
          "tone": "even"
        },
        "real": {
          "n": 112,
          "mean": 0.09570535714285718,
          "lo": -0.10483762808990989,
          "hi": 0.29624834237562425,
          "total": 10.719000000000003,
          "tone": "even"
        },
        "trades": "270 trades",
        "title": "Rebuilding trading with in the hunt: +0.02 expected wins a " +
          "trade, within chance of zero on the day; +0.10 wins a trade, " +
          "within chance of zero since."
      },
      {
        "other": "rebuild",
        "n": 332,
        "now": {
          "n": 332,
          "mean": 0,
          "lo": -0.04873810279410002,
          "hi": 0.04873810279410002,
          "total": 0,
          "tone": "even"
        },
        "real": {
          "n": 108,
          "mean": 0,
          "lo": -0.16733499000474833,
          "hi": 0.16733499000474833,
          "total": 0,
          "tone": "even"
        },
        "trades": "332 trades",
        "title": "Rebuilding trading with rebuilding: +0.00 expected wins a trade, " +
          "within chance of zero on the day; +0.00 wins a trade, within " +
          "chance of zero since."
      }
    ],
    "rows": [
      {
        "fid": "0032",
        "name": "Washington Commanders",
        "n": 526,
        "perSeason": 52.6,
        "tiers": {
          "contender": 0.5570342205323194,
          "hunt": 0.10456273764258556,
          "rebuild": 0.33840304182509506
        },
        "forNow": {
          "k": 129,
          "n": 526,
          "p": 0.24524714828897337,
          "lo": 0.20847935514603883,
          "hi": 0.2820149414319079,
          "base": 0.24356955380577427,
          "tone": "even"
        },
        "forLater": {
          "k": 112,
          "n": 526,
          "p": 0.21292775665399238,
          "lo": 0.17794237330444235,
          "hi": 0.24791314000354242,
          "base": 0.24356955380577427,
          "tone": "even"
        },
        "day": {
          "n": 526,
          "mean": -0.012439163498098862,
          "lo": -0.048166808029144005,
          "hi": 0.023288481032946277,
          "total": -6.543000000000001,
          "tone": "even"
        },
        "since": {
          "n": 197,
          "mean": -0.01694923857868022,
          "lo": -0.12151624444084731,
          "hi": 0.08761776728348687,
          "total": -3.3390000000000035,
          "tone": "even"
        },
        "picks": {
          "n": 526,
          "mean": 0.017110266159695818,
          "lo": -0.12863053966860513,
          "hi": 0.16285107198799678,
          "total": 9,
          "tone": "even"
        },
        "age": {
          "n": 381,
          "mean": 0.009676915385576882,
          "lo": -0.30113878538598815,
          "hi": 0.32049261615714186,
          "total": 3.6869047619047923,
          "tone": "even"
        },
        "partner": {
          "fid": "0027",
          "n": 93
        },
        "partnerName": "Los Angeles Rams",
        "partnersCount": 29,
        "standingTitle": "Contenders 55.7%, In the hunt 10.5%, Rebuilding 33.8%",
        "partnerTrades": "93 trades"
      },
      {
        "fid": "0025",
        "name": "Arizona Cardinals",
        "n": 140,
        "perSeason": 14,
        "tiers": {
          "contender": 0.18571428571428572,
          "hunt": 0.37142857142857144,
          "rebuild": 0.44285714285714284
        },
        "forNow": {
          "k": 32,
          "n": 140,
          "p": 0.22857142857142856,
          "lo": 0.15901282018971927,
          "hi": 0.2981300369531379,
          "base": 0.24356955380577427,
          "tone": "even"
        },
        "forLater": {
          "k": 39,
          "n": 140,
          "p": 0.2785714285714286,
          "lo": 0.20431107575783092,
          "hi": 0.35283178138502624,
          "base": 0.24356955380577427,
          "tone": "even"
        },
        "day": {
          "n": 140,
          "mean": -0.07910714285714285,
          "lo": -0.14243692006577485,
          "hi": -0.015777365648510858,
          "total": -11.075,
          "tone": "down"
        },
        "since": {
          "n": 45,
          "mean": 0.06115555555555556,
          "lo": -0.12188281078988113,
          "hi": 0.24419392190099226,
          "total": 2.7520000000000002,
          "tone": "even"
        },
        "picks": {
          "n": 140,
          "mean": 0.16428571428571428,
          "lo": -0.13017435739285774,
          "hi": 0.4587457859642863,
          "total": 23,
          "tone": "even"
        },
        "age": {
          "n": 56,
          "mean": -0.1396003401360546,
          "lo": -0.9315771326711271,
          "hi": 0.6523764523990179,
          "total": -7.817619047619058,
          "tone": "even"
        },
        "partner": {
          "fid": "0032",
          "n": 28
        },
        "partnerName": "Washington Commanders",
        "partnersCount": 30,
        "standingTitle": "Contenders 18.6%, In the hunt 37.1%, Rebuilding 44.3%",
        "partnerTrades": "28 trades"
      }
    ]
  },
  "provenance": {
    "source": "Lab compiled facts " +
      "sha256:6a9995ad26b4e4c5bc63dc1b771f6c38c3af71af7493d70263f332aed3" +
      "748830",
    "freshness": {
      "state": "stale",
      "fetchedAt": "2026-10-06T15:57:20+00:00",
      "note": "Frozen historical input; not current league truth"
    },
    "kind": "archive"
  },
  "status": "ok",
  "absenceReason": null,
  "count": 3810,
  "threshold": 40,
  "baseline": "The league's own posture and window shares in the same seasons; " +
    "a share is coloured only where it clears chance",
  "cutoff": "2026-10-06T01:42:57+00:00",
  "method": "both sides of every valued trade, cut by standing, window and " +
    "posture; 1.96 standard-error ranges",
  "inputSHA": "6a9995ad26b4e4c5bc63dc1b771f6c38c3af71af7493d70263f332aed3748830",
  "buildStamp": "dev ",
  "howCounted": "A team's tier is its rank on the day of the trade: its live " +
    "standing once three games were final, otherwise the previous " +
    "season's finish, exactly as the frozen archive recorded it. " +
    "Contenders are ranked 1 to 10, teams in the hunt 11 to 22, " +
    "rebuilding teams 23 to 32; the standings run through the archive " +
    "cutoff 2026-10-06, never today's live table. What a trade did is " +
    "read from each side's flow: bought for now when it took in more " +
    "player value than it sent and sent more pick value than it took, " +
    "built for later the reverse, like for like otherwise. Value is " +
    "forecast wins on the day; a flow under 0.05 forecast wins counts " +
    "as even. On the day is the expected wins a side gained at the " +
    "league's own prices; since is the wins its assets actually " +
    "delivered against the other side's, for trades with two or more " +
    "played seasons. Every range is the 95% likely range of the " +
    "average per trade; a range touching zero is within chance, not a " +
    "finding."
};
export const teamWire = {
  "value": {
    "flowScale": 23.68099999999999,
    "fid": "0001",
    "name": "Buffalo Bills",
    "asOf": "Standings as of the archive cutoff 2026-10-06, not today's live standings.",
    "sides": 166,
    "empty": false,
    "note": "",
    "seasons": [
      {
        "season": 2017,
        "n": 2,
        "now": 0,
        "later": 2,
        "swap": 0,
        "against": 0,
        "flagged": false,
        "mostlyText": "in the hunt",
        "net": -0.21000000000000002,
        "finish": 27,
        "label": "2017 · finished 27",
        "title": "2017: 2 trade sides; 0 for now, 2 for later, 0 like for like; finished 27"
      },
      {
        "season": 2023,
        "n": 56,
        "now": 19,
        "later": 10,
        "swap": 27,
        "against": 18,
        "flagged": true,
        "mostlyText": "rebuilding",
        "net": 0.06999999999999978,
        "finish": 3,
        "label": "2023 · finished 3",
        "title": "2023: 56 trade sides; 19 for now, 10 for later, 27 like for " +
          "like; finished 3. 18 went against its standing that day."
      }
    ],
    "seasonSay": "166 trade sides, 18.4 a season. Bars: bought for now, built for " +
      "later, like for like. An outlined season is one where at least a " +
      "third of its trades went against the team's standing that day: " +
      "buying for now while rebuilding, or building while contending.",
    "posture": [
      {
        "id": "now",
        "label": "Bought for now",
        "p": 0.30120481927710846,
        "tone": "even",
        "title": "Bought for now in 30.1% of these trades, within chance of the league's share"
      },
      {
        "id": "later",
        "label": "Built for later",
        "p": 0.26506024096385544,
        "tone": "even",
        "title": "Built for later in 26.5% of these trades, within chance of the league's share"
      },
      {
        "id": "swap",
        "label": "Like for like",
        "p": 0.43373493975903615,
        "tone": "down",
        "title": "Like for like in 43.4% of these trades, below the league's share beyond chance"
      }
    ],
    "kinds": [
      {
        "kind": "WR",
        "name": "Wide receivers",
        "inN": 34,
        "outN": 37,
        "inW": 23.68099999999999,
        "outW": 22.149000000000004,
        "text": "37 out · 34 in"
      },
      {
        "kind": "PK",
        "name": "Kickers",
        "inN": 6,
        "outN": 1,
        "inW": 1.739,
        "outW": 0.276,
        "text": "1 out · 6 in"
      }
    ],
    "kindSay": "Bars are forecast wins sent and taken in for each kind of asset; " +
      "the figures count the assets.",
    "partners": [
      {
        "fid": "0030",
        "name": "New York Giants",
        "n": 27,
        "net": {
          "n": 27,
          "mean": -0.06255555555555556,
          "lo": -0.2740902515608565,
          "hi": 0.14897914044974536,
          "total": -1.689,
          "tone": "even"
        },
        "tradesText": "27 trades",
        "netText": "-1.69 on the day"
      },
      {
        "fid": "0032",
        "name": "Washington Commanders",
        "n": 18,
        "net": {
          "n": 18,
          "mean": 0.049777777777777775,
          "lo": -0.0970406760814333,
          "hi": 0.19659623163698886,
          "total": 0.8959999999999999,
          "tone": "even"
        },
        "tradesText": "18 trades",
        "netText": "+0.90 on the day"
      }
    ],
    "partnersSay": "And 14 more partners in view.",
    "behaviour": {
      "n": 166,
      "perSeason": 18.444444444444443,
      "posture": {
        "later": {
          "k": 44,
          "n": 166,
          "p": 0.26506024096385544,
          "lo": 0.1979173258520652,
          "hi": 0.33220315607564566,
          "base": 0.24356955380577427,
          "tone": "even"
        },
        "now": {
          "k": 50,
          "n": 166,
          "p": 0.30120481927710846,
          "lo": 0.23141245091830248,
          "hi": 0.37099718763591444,
          "base": 0.24356955380577427,
          "tone": "even"
        },
        "swap": {
          "k": 72,
          "n": 166,
          "p": 0.43373493975903615,
          "lo": 0.35834313941610996,
          "hi": 0.5091267401019623,
          "base": 0.5128608923884515,
          "tone": "down"
        }
      },
      "partners": {
        "contender": {
          "k": 65,
          "n": 166,
          "p": 0.39156626506024095,
          "lo": 0.3173137267527322,
          "hi": 0.4658188033677497,
          "tone": "even"
        },
        "hunt": {
          "k": 37,
          "n": 166,
          "p": 0.22289156626506024,
          "lo": 0.15957903801604245,
          "hi": 0.286204094514078,
          "tone": "even"
        },
        "rebuild": {
          "k": 64,
          "n": 166,
          "p": 0.3855421686746988,
          "lo": 0.3114991677499975,
          "hi": 0.45958516959940016,
          "tone": "even"
        }
      },
      "windows": {
        "season": {
          "k": 28,
          "n": 166,
          "p": 0.1686746987951807,
          "lo": 0.11170912212383927,
          "hi": 0.22564027546652216,
          "base": 0.2587926509186352,
          "tone": "down"
        },
        "spring": {
          "k": 88,
          "n": 166,
          "p": 0.5301204819277109,
          "lo": 0.45419587071268663,
          "hi": 0.6060450931427351,
          "base": 0.4146981627296588,
          "tone": "up"
        },
        "summer": {
          "k": 50,
          "n": 166,
          "p": 0.30120481927710846,
          "lo": 0.23141245091830248,
          "hi": 0.37099718763591444,
          "base": 0.32650918635170606,
          "tone": "even"
        }
      },
      "now": {
        "n": 166,
        "mean": 0.0037710843373493963,
        "lo": -0.06709225139027299,
        "hi": 0.07463442006497177,
        "total": 0.6259999999999998,
        "tone": "even"
      },
      "real": {
        "n": 55,
        "mean": 0.05341818181818182,
        "lo": -0.14163149923596435,
        "hi": 0.24846786287232803,
        "total": 2.938,
        "tone": "even"
      },
      "picks": {
        "n": 166,
        "mean": -0.06626506024096386,
        "lo": -0.34957444205983734,
        "hi": 0.21704432157790965,
        "total": -11,
        "tone": "even"
      },
      "age": {
        "n": 96,
        "mean": 0.5007142857142854,
        "lo": -0.019649321146368304,
        "hi": 1.0210778925749393,
        "total": 48.0685714285714,
        "tone": "even"
      }
    },
    "asTier": [
      {
        "tier": "contender",
        "n": 84,
        "edge": {
          "n": 84,
          "mean": -0.015119047619047617,
          "lo": -0.11031537294776517,
          "hi": 0.08007727770966994,
          "total": -1.2699999999999998,
          "tone": "even"
        },
        "label": "As contender (84)",
        "text": "-0.02 expected wins a trade, within chance of zero"
      },
      {
        "tier": "hunt",
        "n": 10,
        "edge": {
          "n": 10,
          "mean": 0.0506,
          "lo": -0.14255724589739485,
          "hi": 0.24375724589739486,
          "total": 0.506,
          "tone": "even"
        },
        "label": "As in the hunt (10)",
        "text": "+0.05 expected wins a trade, within chance of zero"
      },
      {
        "tier": "rebuild",
        "n": 72,
        "edge": {
          "n": 72,
          "mean": 0.019305555555555555,
          "lo": -0.09832569241937286,
          "hi": 0.13693680353048399,
          "total": 1.39,
          "tone": "even"
        },
        "label": "As rebuilding (72)",
        "text": "+0.02 expected wins a trade, within chance of zero"
      }
    ],
    "day": {
      "n": 166,
      "mean": 0.0037710843373493963,
      "lo": -0.06709225139027299,
      "hi": 0.07463442006497177,
      "total": 0.6259999999999998,
      "tone": "even"
    },
    "since": {
      "n": 55,
      "mean": 0.05341818181818182,
      "lo": -0.14163149923596435,
      "hi": 0.24846786287232803,
      "total": 2.938,
      "tone": "even"
    },
    "dayText": "+0.63 expected wins on the day, all trades",
    "sinceText": "+2.94 wins since, over 55 trades with 2+ seasons played"
  },
  "provenance": {
    "source": "Lab compiled facts " +
      "sha256:6a9995ad26b4e4c5bc63dc1b771f6c38c3af71af7493d70263f332aed3" +
      "748830",
    "freshness": {
      "state": "stale",
      "fetchedAt": "2026-10-06T15:57:20+00:00",
      "note": "Frozen historical input; not current league truth"
    },
    "kind": "archive"
  },
  "status": "ok",
  "absenceReason": null,
  "count": 166,
  "threshold": 1,
  "baseline": "The league's own posture and window shares in the same seasons; " +
    "a share is coloured only where it clears chance",
  "cutoff": "2026-10-06T01:42:57+00:00",
  "method": "this team's sides of the valued trades against the league in the " +
    "same seasons; 1.96 standard-error ranges",
  "inputSHA": "6a9995ad26b4e4c5bc63dc1b771f6c38c3af71af7493d70263f332aed3748830",
  "buildStamp": "dev ",
  "howCounted": "A team's tier is its rank on the day of the trade: its live " +
    "standing once three games were final, otherwise the previous " +
    "season's finish, exactly as the frozen archive recorded it. " +
    "Contenders are ranked 1 to 10, teams in the hunt 11 to 22, " +
    "rebuilding teams 23 to 32; the standings run through the archive " +
    "cutoff 2026-10-06, never today's live table. What a trade did is " +
    "read from each side's flow: bought for now when it took in more " +
    "player value than it sent and sent more pick value than it took, " +
    "built for later the reverse, like for like otherwise. Value is " +
    "forecast wins on the day; a flow under 0.05 forecast wins counts " +
    "as even. On the day is the expected wins a side gained at the " +
    "league's own prices; since is the wins its assets actually " +
    "delivered against the other side's, for trades with two or more " +
    "played seasons. Every range is the 95% likely range of the " +
    "average per trade; a range touching zero is within chance, not a " +
    "finding."
};
export const thinWire = {
  "value": {
    "summary": "6 trade sides of valued trades, 2017–2017. On the day is " +
      "expected wins gained at the league's own prices; since is wins " +
      "actually delivered, for trades with two or more seasons played.",
    "asOf": "Standings as of the archive cutoff 2026-10-06, not today's live standings.",
    "headline": "",
    "note": "Each trade has two sides, one per team. A team's tier is its " +
      "rank that day: live once three games are final, else last " +
      "season's finish, as the archive recorded it.",
    "picks": [
      {
        "id": "contender",
        "label": "Contenders",
        "hint": "Ranked 1 to 10 on the day of the trade",
        "count": 9
      },
      {
        "id": "hunt",
        "label": "In the hunt",
        "hint": "Ranked 11 to 22 on the day of the trade",
        "count": 6
      },
      {
        "id": "rebuild",
        "label": "Rebuilding",
        "hint": "Ranked 23 to 32 on the day of the trade",
        "count": 7
      }
    ],
    "teams": [
      {
        "fid": "0025",
        "name": "Arizona Cardinals"
      },
      {
        "fid": "0017",
        "name": "Atlanta Falcons"
      },
      {
        "fid": "0013",
        "name": "Baltimore Ravens"
      },
      {
        "fid": "0001",
        "name": "Buffalo Bills"
      },
      {
        "fid": "0018",
        "name": "Carolina Panthers"
      },
      {
        "fid": "0021",
        "name": "Chicago Bears"
      },
      {
        "fid": "0014",
        "name": "Cincinnati Bengals"
      },
      {
        "fid": "0015",
        "name": "Cleveland Browns"
      },
      {
        "fid": "0029",
        "name": "Dallas Cowboys"
      },
      {
        "fid": "0008",
        "name": "Denver Broncos"
      },
      {
        "fid": "0022",
        "name": "Detroit Lions"
      },
      {
        "fid": "0023",
        "name": "Green Bay Packers"
      },
      {
        "fid": "0009",
        "name": "Houston Texans"
      },
      {
        "fid": "0010",
        "name": "Indianapolis Colts"
      },
      {
        "fid": "0011",
        "name": "Jacksonville Jaguars"
      },
      {
        "fid": "0006",
        "name": "Kansas City Chiefs"
      },
      {
        "fid": "0007",
        "name": "Las Vegas Raiders"
      },
      {
        "fid": "0005",
        "name": "Los Angeles Chargers"
      },
      {
        "fid": "0027",
        "name": "Los Angeles Rams"
      },
      {
        "fid": "0004",
        "name": "Miami Dolphins"
      },
      {
        "fid": "0024",
        "name": "Minnesota Vikings"
      },
      {
        "fid": "0002",
        "name": "New England Patriots"
      },
      {
        "fid": "0019",
        "name": "New Orleans Saints"
      },
      {
        "fid": "0030",
        "name": "New York Giants"
      },
      {
        "fid": "0003",
        "name": "New York Jets"
      },
      {
        "fid": "0031",
        "name": "Philadelphia Eagles"
      },
      {
        "fid": "0016",
        "name": "Pittsburgh Steelers"
      },
      {
        "fid": "0026",
        "name": "San Francisco 49ers"
      },
      {
        "fid": "0028",
        "name": "Seattle Seahawks"
      },
      {
        "fid": "0020",
        "name": "Tampa Bay Buccaneers"
      },
      {
        "fid": "0012",
        "name": "Tennessee Titans"
      },
      {
        "fid": "0032",
        "name": "Washington Commanders"
      }
    ],
    "cards": null,
    "trendSay": null,
    "trends": {
      "seasons": null,
      "tiers": null
    },
    "pairs": null,
    "rows": null
  },
  "provenance": {
    "source": "Lab compiled facts " +
      "sha256:6a9995ad26b4e4c5bc63dc1b771f6c38c3af71af7493d70263f332aed3" +
      "748830",
    "freshness": {
      "state": "stale",
      "fetchedAt": "2026-10-06T15:57:20+00:00",
      "note": "Frozen historical input; not current league truth"
    },
    "kind": "archive"
  },
  "status": "thin",
  "absenceReason": "Fewer than 40 trade sides in view; widen the standing, time of " +
    "year or what the trades did",
  "count": 6,
  "threshold": 40,
  "baseline": "The league's own posture and window shares in the same seasons; " +
    "a share is coloured only where it clears chance",
  "cutoff": "2026-10-06T01:42:57+00:00",
  "method": "both sides of every valued trade, cut by standing, window and " +
    "posture; 1.96 standard-error ranges",
  "inputSHA": "6a9995ad26b4e4c5bc63dc1b771f6c38c3af71af7493d70263f332aed3748830",
  "buildStamp": "dev ",
  "howCounted": "A team's tier is its rank on the day of the trade: its live " +
    "standing once three games were final, otherwise the previous " +
    "season's finish, exactly as the frozen archive recorded it. " +
    "Contenders are ranked 1 to 10, teams in the hunt 11 to 22, " +
    "rebuilding teams 23 to 32; the standings run through the archive " +
    "cutoff 2026-10-06, never today's live table. What a trade did is " +
    "read from each side's flow: bought for now when it took in more " +
    "player value than it sent and sent more pick value than it took, " +
    "built for later the reverse, like for like otherwise. Value is " +
    "forecast wins on the day; a flow under 0.05 forecast wins counts " +
    "as even. On the day is the expected wins a side gained at the " +
    "league's own prices; since is the wins its assets actually " +
    "delivered against the other side's, for trades with two or more " +
    "played seasons. Every range is the 95% likely range of the " +
    "average per trade; a range touching zero is within chance, not a " +
    "finding."
};
export const emptyWire = {
  "value": {
    "flowScale": 0.01,
    "fid": "0001",
    "name": "Buffalo Bills",
    "asOf": "Standings as of the archive cutoff 2026-10-06, not today's live standings.",
    "sides": 0,
    "empty": true,
    "note": "This team shows no valued trades in view. Widen the seasons, " +
      "time of year or what the trades did.",
    "seasons": [
      {
        "season": 2017,
        "n": 0,
        "now": 0,
        "later": 0,
        "swap": 0,
        "against": 0,
        "flagged": false,
        "mostlyText": "",
        "net": null,
        "finish": 27,
        "label": "2017 · finished 27",
        "title": "2017: 0 trade sides; 0 for now, 0 for later, 0 like for like; finished 27"
      }
    ],
    "seasonSay": "0 trade sides, 0.0 a season. Bars: bought for now, built for " +
      "later, like for like. An outlined season is one where at least a " +
      "third of its trades went against the team's standing that day: " +
      "buying for now while rebuilding, or building while contending.",
    "posture": [
      {
        "id": "now",
        "label": "Bought for now",
        "p": null,
        "tone": "",
        "title": ""
      },
      {
        "id": "later",
        "label": "Built for later",
        "p": null,
        "tone": "",
        "title": ""
      },
      {
        "id": "swap",
        "label": "Like for like",
        "p": null,
        "tone": "",
        "title": ""
      }
    ],
    "kinds": [],
    "kindSay": "Bars are forecast wins sent and taken in for each kind of asset; " +
      "the figures count the assets.",
    "partners": [],
    "partnersSay": "No partners in view.",
    "behaviour": {
      "n": 0,
      "perSeason": 0,
      "posture": {
        "later": {
          "k": 0,
          "n": 0,
          "p": null
        },
        "now": {
          "k": 0,
          "n": 0,
          "p": null
        },
        "swap": {
          "k": 0,
          "n": 0,
          "p": null
        }
      },
      "partners": {
        "contender": {
          "k": 0,
          "n": 0,
          "p": null
        },
        "hunt": {
          "k": 0,
          "n": 0,
          "p": null
        },
        "rebuild": {
          "k": 0,
          "n": 0,
          "p": null
        }
      },
      "windows": {
        "season": {
          "k": 0,
          "n": 0,
          "p": null
        },
        "spring": {
          "k": 0,
          "n": 0,
          "p": null
        },
        "summer": {
          "k": 0,
          "n": 0,
          "p": null
        }
      },
      "now": {
        "n": 0,
        "mean": null,
        "lo": null,
        "hi": null,
        "tone": "even"
      },
      "real": {
        "n": 0,
        "mean": null,
        "lo": null,
        "hi": null,
        "tone": "even"
      },
      "picks": {
        "n": 0,
        "mean": null,
        "lo": null,
        "hi": null,
        "tone": "even"
      },
      "age": {
        "n": 0,
        "mean": null,
        "lo": null,
        "hi": null,
        "tone": "even"
      }
    },
    "asTier": [
      {
        "tier": "contender",
        "n": 0,
        "edge": {
          "n": 0,
          "mean": null,
          "lo": null,
          "hi": null,
          "tone": "even"
        },
        "label": "As contender (0)",
        "text": "—"
      },
      {
        "tier": "hunt",
        "n": 0,
        "edge": {
          "n": 0,
          "mean": null,
          "lo": null,
          "hi": null,
          "tone": "even"
        },
        "label": "As in the hunt (0)",
        "text": "—"
      },
      {
        "tier": "rebuild",
        "n": 0,
        "edge": {
          "n": 0,
          "mean": null,
          "lo": null,
          "hi": null,
          "tone": "even"
        },
        "label": "As rebuilding (0)",
        "text": "—"
      }
    ],
    "day": {
      "n": 0,
      "mean": null,
      "lo": null,
      "hi": null,
      "tone": "even"
    },
    "since": {
      "n": 0,
      "mean": null,
      "lo": null,
      "hi": null,
      "tone": "even"
    },
    "dayText": "Too few trades for a total on the day",
    "sinceText": ""
  },
  "provenance": {
    "source": "Lab compiled facts " +
      "sha256:6a9995ad26b4e4c5bc63dc1b771f6c38c3af71af7493d70263f332aed3" +
      "748830",
    "freshness": {
      "state": "stale",
      "fetchedAt": "2026-10-06T15:57:20+00:00",
      "note": "Frozen historical input; not current league truth"
    },
    "kind": "archive"
  },
  "status": "thin",
  "absenceReason": "No trade sides for this team in the seasons and filters in view",
  "count": 0,
  "threshold": 1,
  "baseline": "The league's own posture and window shares in the same seasons; " +
    "a share is coloured only where it clears chance",
  "cutoff": "2026-10-06T01:42:57+00:00",
  "method": "this team's sides of the valued trades against the league in the " +
    "same seasons; 1.96 standard-error ranges",
  "inputSHA": "6a9995ad26b4e4c5bc63dc1b771f6c38c3af71af7493d70263f332aed3748830",
  "buildStamp": "dev ",
  "howCounted": "A team's tier is its rank on the day of the trade: its live " +
    "standing once three games were final, otherwise the previous " +
    "season's finish, exactly as the frozen archive recorded it. " +
    "Contenders are ranked 1 to 10, teams in the hunt 11 to 22, " +
    "rebuilding teams 23 to 32; the standings run through the archive " +
    "cutoff 2026-10-06, never today's live table. What a trade did is " +
    "read from each side's flow: bought for now when it took in more " +
    "player value than it sent and sent more pick value than it took, " +
    "built for later the reverse, like for like otherwise. Value is " +
    "forecast wins on the day; a flow under 0.05 forecast wins counts " +
    "as even. On the day is the expected wins a side gained at the " +
    "league's own prices; since is the wins its assets actually " +
    "delivered against the other side's, for trades with two or more " +
    "played seasons. Every range is the 95% likely range of the " +
    "average per trade; a range touching zero is within chance, not a " +
    "finding."
};
