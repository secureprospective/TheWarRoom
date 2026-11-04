# Coverage soak report — run-1104-0952

Walk loops completed: 46. Statements: 58,068 total, 19,376 ran (33.4%), 38,692 never ran.

Never-ran ≠ unused. Review each package: in a mock / on no screen / ruled out.

| Package | Stmts | Ran | Never ran | % ran | Note |
|---|---:|---:|---:|---:|---|
| internal/commissioner | 10,572 | 3,914 | 6,658 | 37% |  |
| internal/store/history | 5,042 | 2,260 | 2,782 | 45% |  |
| internal/app/mflref/contracts | 1,555 | 73 | 1,482 | 5% |  |
| internal/facts | 1,373 | 0 | 1,373 | 0% |  |
| internal/commissioner/market | 1,452 | 95 | 1,357 | 7% |  |
| internal/lab | 1,340 | 143 | 1,197 | 11% | Lab, ruled out 10-26 |
| internal/app/now | 1,256 | 236 | 1,020 | 19% |  |
| internal/store/state | 1,271 | 289 | 982 | 23% |  |
| internal/serverhttp | 929 | 0 | 929 | 0% |  |
| internal/composition | 1,805 | 918 | 887 | 51% |  |
| internal/nativecheck | 1,055 | 266 | 789 | 25% |  |
| internal/leagueservice | 1,118 | 374 | 744 | 33% |  |
| internal/mflref/envelope | 727 | 4 | 723 | 1% |  |
| internal/serverclient | 722 | 0 | 722 | 0% |  |
| internal/leaguemigration/mflsource | 1,021 | 305 | 716 | 30% |  |
| internal/db | 1,223 | 532 | 691 | 43% |  |
| internal/app/sources | 976 | 286 | 690 | 29% |  |
| internal/app/season | 661 | 7 | 654 | 1% |  |
| internal/app/mflref/trades | 624 | 15 | 609 | 2% |  |
| internal/app/research | 1,317 | 722 | 595 | 55% | Research, split out 10-28 |
| internal/app/offers | 551 | 6 | 545 | 1% |  |
| internal/governance | 622 | 80 | 542 | 13% |  |
| internal/leaguemigration | 671 | 144 | 527 | 21% |  |
| internal/store/accounts | 560 | 54 | 506 | 10% |  |
| internal/leaguerules | 1,384 | 885 | 499 | 64% |  |
| internal/app/control | 923 | 432 | 491 | 47% |  |
| internal/accounts | 498 | 14 | 484 | 3% |  |
| internal/ingestion/livegame | 591 | 143 | 448 | 24% |  |
| internal/transactions | 446 | 3 | 443 | 1% |  |
| internal/fantasyscore/source | 1,183 | 758 | 425 | 64% |  |
| internal/engine/rookiedraft | 451 | 36 | 415 | 8% |  |
| internal/engine/competition | 669 | 256 | 413 | 38% |  |
| internal/app/market | 587 | 241 | 346 | 41% |  |
| internal/commandcatalog | 770 | 425 | 345 | 55% |  |
| internal/app/platform | 479 | 174 | 305 | 36% |  |
| internal/app/draft | 303 | 4 | 299 | 1% |  |
| internal/app/mflref/planwatch | 343 | 51 | 292 | 15% |  |
| internal/ingestion/leaguearchive | 266 | 0 | 266 | 0% |  |
| internal/releasetrust | 264 | 0 | 264 | 0% |  |
| internal/app/comms | 234 | 9 | 225 | 4% |  |
| internal/app/core | 363 | 138 | 225 | 38% |  |
| internal/store/mflref/moves | 248 | 27 | 221 | 11% |  |
| internal/engine/race | 214 | 0 | 214 | 0% |  |
| internal/store/rulebook | 320 | 120 | 200 | 38% |  |
| internal/fantasyscore | 359 | 162 | 197 | 45% |  |
| internal/app/commissioner | 229 | 44 | 185 | 19% |  |
| internal/store/personal | 222 | 37 | 185 | 17% |  |
| internal/ingestion/livestats | 152 | 1 | 151 | 1% |  |
| internal/headshot | 153 | 9 | 144 | 6% |  |
| internal/app/commissioner/native | 224 | 82 | 142 | 37% |  |
| internal/snapshot | 278 | 139 | 139 | 50% |  |
| internal/measures | 294 | 158 | 136 | 54% |  |
| internal/manifest | 129 | 0 | 129 | 0% |  |
| internal/ingestion | 288 | 162 | 126 | 56% |  |
| internal/mflref/bid | 115 | 0 | 115 | 0% |  |
| internal/appvalue | 538 | 426 | 112 | 79% |  |
| internal/ingestion/researchfeeds | 232 | 120 | 112 | 52% |  |
| internal/transactions/contracts | 111 | 0 | 111 | 0% |  |
| internal/transactions/deadcap | 116 | 10 | 106 | 9% |  |
| internal/app/report | 174 | 78 | 96 | 45% |  |
| internal/engine/contractacts | 93 | 0 | 93 | 0% |  |
| internal/app/mflref/bids | 97 | 6 | 91 | 6% |  |
| internal/store/params | 203 | 125 | 78 | 62% |  |
| internal/mfl | 237 | 161 | 76 | 68% |  |
| internal/lineupdelta | 89 | 20 | 69 | 22% |  |
| internal/app/offers/pool | 85 | 22 | 63 | 26% |  |
| internal/app/scoring | 212 | 152 | 60 | 72% |  |
| internal/app/archive | 58 | 0 | 58 | 0% |  |
| internal/leagueclock | 178 | 120 | 58 | 67% |  |
| internal/app/mflref/seed | 53 | 0 | 53 | 0% |  |
| internal/modelrun | 286 | 236 | 50 | 83% |  |
| internal/serverversion | 50 | 0 | 50 | 0% |  |
| internal/ingestion/college | 48 | 0 | 48 | 0% |  |
| internal/ingestion/pendingtrades | 76 | 29 | 47 | 38% |  |
| internal/ingestion/injuries | 74 | 28 | 46 | 38% |  |
| internal/leaguefeed | 76 | 33 | 43 | 43% |  |
| internal/engine/bid | 42 | 0 | 42 | 0% |  |
| internal/ingestion/league | 100 | 58 | 42 | 58% |  |
| internal/app/race | 119 | 78 | 41 | 66% |  |
| internal/ingestion/nflschedule | 95 | 54 | 41 | 57% |  |
| internal/mflkey | 52 | 15 | 37 | 29% |  |
| internal/app/mflref/migration | 132 | 96 | 36 | 73% |  |
| internal/ingestion/draftreports | 123 | 88 | 35 | 72% |  |
| internal/app/offers/personal | 33 | 0 | 33 | 0% |  |
| internal/leaguerecovery | 32 | 0 | 32 | 0% |  |
| internal/m2service | 211 | 179 | 32 | 85% |  |
| internal/ingestion/livescoring | 108 | 77 | 31 | 71% |  |
| internal/ingestion/sitenews | 31 | 0 | 31 | 0% |  |
| internal/ingestion/marketfeeds | 126 | 97 | 29 | 77% |  |
| internal/store/mflref/tradeplans | 29 | 0 | 29 | 0% |  |
| internal/model | 371 | 343 | 28 | 92% |  |
| internal/mflref/lineup | 100 | 73 | 27 | 73% |  |
| internal/offline | 57 | 30 | 27 | 53% |  |
| internal/archive | 96 | 71 | 25 | 74% |  |
| internal/playerrecord | 128 | 104 | 24 | 81% |  |
| internal/domain | 81 | 58 | 23 | 72% |  |
| internal/ingestion/playoffs | 76 | 53 | 23 | 70% |  |
| internal/store/playerrecord | 103 | 81 | 22 | 79% |  |
| internal/ingestion/transactions | 70 | 49 | 21 | 70% |  |
| internal/ingestion/crosswalk | 111 | 91 | 20 | 82% |  |
| internal/ingestion/weeklyresults | 89 | 69 | 20 | 78% |  |
| internal/mflref/whatif | 20 | 0 | 20 | 0% |  |
| internal/transactions/freeagency | 19 | 0 | 19 | 0% |  |
| internal/playerid | 46 | 29 | 17 | 63% |  |
| internal/normalize | 100 | 84 | 16 | 84% |  |
| internal/ingestion/playerscores | 47 | 33 | 14 | 70% |  |
| internal/ingestion/players | 36 | 23 | 13 | 64% |  |
| internal/ingestion/feeds | 79 | 67 | 12 | 85% |  |
| internal/ingestion/rosters | 35 | 23 | 12 | 66% |  |
| internal/powerrankings | 153 | 141 | 12 | 92% |  |
| internal/ingestion/leaguestandings | 43 | 32 | 11 | 74% |  |
| internal/pickvalue | 128 | 117 | 11 | 91% |  |
| internal/ingestion/leagueschedule | 39 | 29 | 10 | 74% |  |
| internal/engine | 19 | 10 | 9 | 53% |  |
| internal/ingestion/freeagents | 37 | 28 | 9 | 76% |  |
| internal/leagueweek | 54 | 45 | 9 | 83% |  |
| internal/engine/eligibility | 52 | 44 | 8 | 85% |  |
| internal/ingestion/salaryadjustments | 26 | 18 | 8 | 69% |  |
| internal/transactions/acquisitions | 7 | 0 | 7 | 0% |  |
| internal/feedwire | 19 | 14 | 5 | 74% |  |
| internal/projection | 41 | 36 | 5 | 88% |  |
| internal/ingestion/contracts | 37 | 33 | 4 | 89% |  |
| internal/numeric | 7 | 6 | 1 | 86% |  |
| github.com/secureprospective/TheWarRoom | 1 | 1 | 0 | 100% |  |

## Functions that never ran, by package (top 40 packages by never-ran statements)

- **internal/commissioner** (527): Action, ActiveCount, ActiveFor, AllowDebtReduction, AuthorizeFranchiseOwner, BidPlayerOf, ChangedPreviewFields, CheckContracts, CheckHistoryLinkAppend, CheckNativeRecoveryAppend, CheckTokenFranchise, Claim, Clone, CloseBooks, ClosedBooksHash, CommandPreview, CommandSafeguards, ComparablePresets, CompleteWaiverOrder, Counts, Decode, DecodeBracketChange, DecodeHistoryLink, DecodeLedgerFilter, DecodePresetApply, DecodePreview, DecodeScoreChange, DecodeStandingsChange, DecodeTitleChange, DraftPreview, DraftSetupFields, DraftTrayCommand, Exceptions, Final, Find, Franchise, FranchiseHistoryReading, FullFolds, Gate, LastBidOf, Latest, LeaderOf, League, LeagueSides, LineupCorrectionPreview, Live, LiveBids, MFLImportPreset, MatchesRetry, MoveTradeAssets, NamePreviewFields, NativeActionGate, NativeLineupCheckReading, NativeLineupReading, NativePlayerScoreReading, NativePlayerScoreWith, NativeRosterCheckReading, NativeSeedCommand, NativeSeedRetry, NativeStart …
- **internal/store/history** (196): Allow, Allow, Allow, Allow, Allow, AppendEvent, Authority, BidEnvironment, BidEnvironment, Body, CanonicalPlayerWeekStats, CaptureTransfer, CaptureTransferSnapshot, CheckDigest, CheckTransferCommissioner, CheckTransferDestination, CommissionerLedger, CommittedHead, DecideDestination, DecideTransfer, Digest, Disagreements, Error, ExecuteCommissionerBatch, ExecuteCommissionerWithFacts, ExecuteRulesBatchWithFacts, HeldBodies, Identities, ImportDocument, ImportFootball, ImportFootballWeek, Is, LineGames, LineupDelta, LineupFacts, LoadFailed, LoadFootball, ManifestBySHA, MeasuresHeld, Name, NativeBids, NativeLineupCheck, NativeLineupDelta, NativePlayerScore, NativeReserveCheck, NativeRosterCheck, NativeScorePreview, NativeSeedRetry, NativeWaivers, Now, PickValueDrafts, PlayerFacts, PlayerWeekStats, PlayerWeekStats, PlayersWithData, PreviewCommissioner, PreviewCommissionerWithFacts, PublishManifest, RecordDraftPick, RecordImport …
- **internal/app/mflref/contracts** (128): ActFacts, Adjustments, CapReason, Change, ChargeNote, Choice, Choices, Commands, Committed, Competing, ContractChanged, DescribeAct, DescribeWhatIf, Evidence, EvidenceReasons, Figure, Figures, InjuredReserve, IntentLabel, LineupPlan, LoadWhatIf, MirrorWhatIf, Moves, Name, NeedsInvalidation, NeedsValidation, ObservedCharge, OpenWhatIf, ParseScorePeriod, Plan, Player, Position, PositionLimit, Reading, ReleaseBuyoutCount, ReleaseFacts, RosterReports, RosterSize, Rule, ScratchWhatIf, StateLabel, Subject, SupersedeMoves, TargetAcquisition, TargetCheckLineup, TargetContractActs, TargetContracts, TargetDraftLineup, TargetPlayerContract, TargetReleasePlans, TargetRosterReports, TargetWhatIf, TaxiSquad, Terms, UnreadableRules, Validate, Validate, Validate, Validate, Validate …
- **internal/facts** (132): Check, Compile, Fingerprint, Len, Less, ParseCurated, ParseExceptions, Swap, Validate, Validate, Validate, Views, add, add, addNote, addTradeMoves, allPlay, allPlay, allPlayWeeks, asciiString, atoi, birthdate, birthdates, calendar, cellBytes, changed, cmpInt, coalesce, compareCell, contracts, copyPointer, day, decodeAsset, decodeRaw, decodeValue, differ, draft, draftUsers, dump, easternDay, farApart, fill, find, floatText, franchiseNames, get, idList, idSet, indexRows, injuries, intText, isOffer, isRosterType, keyID, lastKickoffs, league, lineup, lineupScores, mergePlayer, mflFigures …
- **internal/commissioner/market** (102): BidOffseasonAt, CheckDueBid, ContractActionOn, NativeBidsReading, NativeMovesReading, NativeTradesReading, NativeWaiversReading, OwnerContractWindow, PrepareDueReserve, QuoteContract, RecordedStandingsOrder, Validate, Validate, act, addCap, appendLanded, appendOwnWaiverClaims, appendPendingBids, appendPendingWaivers, appendWaitingTrades, auctionOpen, award, award, awardPriority, beginReview, bidBar, bidClose, capCounts, checkContract, checkTradeAssets, chooseAward, claim, clearsWaivers, cut, decide, decide, decide, decidedBidWords, describeAssets, eligibleBids, eligibleClaims, experienceYears, forPlayer, franchiseCapInputs, franchiseOf, hasTradeVote, judge, landBid, landClaim, landedPlayer, lateWords, legal, lineCardsHeld, name, name, nativeBidsFrame, nativeFranchiseCap, nativeMove, nativeTradesFrame, nativeWaiversFrame …
- **internal/lab** (111): AgeOf, AssetOf, AssetProfile, Behaviour, BySeason, CheckExceptions, Cov, DeliveredRatio, DraftBuy, DraftNFLRounds, DraftScope, DraftSegments, FilterSides, FitPrices, Flagged, FrozenGaps, GroupOf, HitRate, InScope, Inputs, LeagueMarket, MeanVar, NFLOf, NoteReadings, OfferPhases, OfferStretches, PerDelivered, PickClock, PlaysLike, PositionDraft, SegmentOf, SideYears, SidesInScope, SlotCurve, StretchPrices, TeamPostures, TeamProfile, TeamRows, TeamTiers, TierOf, TierPairs, TierTrends, TradesByPhase, ValuedTrades, VsSlot, ageBand, average, basicRows, bestTimes, classifier, clearsHalf, compareFloat, comparison, costRows, countOfferStretch, curated, decodeGaps, decodeInputs, decorateRow, deliveredRatio …
- **internal/app/now** (101): AttachReportEntities, Bodies, Build, Commands, GamesPerSeason, HeldMatchupStarters, InspectLiveBox, Keep, KnownFranchise, LiveWeek, LiveWeekUnplayed, MatchSpine, MatchupStarterIDs, Possessive, RaceSnapshot, RefreshNow, RefreshSelectedWeek, RefreshWeekAllowed, SeedLiveGames, SelectedRefreshWeek, SpinePlayers, TargetNow, TargetNowRefresh, Validate, address, allSides, beatText, boxRow, boxScore, byeTeams, chooseNowWeek, classicRows, clonePlay, directoryNote, favored, findGame, formCell, game, gameLines, gameRank, gameResult, gameState, hundredths, inspectable, isMine, kickedOff, label, lastThree, liveScoring, matchups, mergeLiveSleeper, mineFirst, myGame, myMatchup, myPlayers, name, needMine, newNowView, newRaceView, nflBoxScore …
- **internal/store/state** (111): ActiveBuyoutLockout, AddCapRelief, AddDeadCap, Adjustments, Adjustments, AppendCalendarEvent, AppendExtensionYears, AppendPhaseTransition, AppendSigningWindow, AppendTradeDeadline, ApplyContract, ApplyContract, CapUsed, CurrentPhase, CurrentStatus, CurrentStatus, DuePlannedEvents, Feed, FranchiseState, FranchiseState, Franchises, FreeAgents, IncOpCount, LedgerCells, Load, LogTradeNote, MoveCellMoney, MovePlayer, MovePlayer, NewWhatIfRecords, NewerThan, OpCount, PaidCells, Player, Player, Reader, RecordStatus, ReleasePlayer, ReleaseStamps, Replace, RolloverSeason, Roster, Rosters, Rosters, Save, Season, SetCell, SetRosterStatus, SetRosterStatus, SignContract, SigningWindowClosed, Stamps, TradeDeadlinePassed, VoidCells, WriteTx, adjustCell, adoptWhatIf, appendDirective, appliedStateVersions, cacheTables …
- **internal/serverhttp** (96): Admit, Admit, CheckTokenAccess, CheckTokenScope, EnrollmentHandler, Error, GoString, GoString, GoString, GoString, GoString, GoString, GoString, GoString, GoString, Handler, ListenLoopback, NewConfirmations, ServeFixture, ServeFixtureChanges, ServeHosted, String, String, String, String, String, String, String, String, String, Validate, Validate, Validate, Validate, Validate, Validate, Validate, Validate, Validate, Validate, Validate, Validate, Validate, Validate, Validate, admitLeagueAccess, agent, agentToken, allow, authenticator, backupAdmission, boundedServer, changed, check, clientAddress, confirm, decodeHosted, decodeNative, dispatch, dispatchOwner …
- **internal/composition** (220): AcquisitionCorrection, AcquisitionSnapshot, AcquisitionTeams, AgeArcSource, AssetInput, BidEnvironment, BrowserOpenURL, CacheScopes, CaptureNativeSeed, CheckMoves, ClearCache, CloseFinishedDrafts, CommandRegistry, CommissionerBackupNow, CommissionerPreviewRegistry, CommissionerSeat, CommissionerSeatSignIn, DeleteKey, DraftDestination, DraftNext, DraftPlayers, DraftRosterMove, DraftTradeAccept, DraftValues, Fetch, FetchRuleNames, FetchSiteNews, HandOff, Injuries, KeyUnavailable, KnownFranchise, Migrate, MirrorLineupDelta, Moves, NowInput, NowRefresh, OffersAssetValues, OffersBuildStamp, OffersCapRoom, OffersDirectory, OffersEngineValues, OffersInjuries, OffersLog, OffersMarketFacts, OffersRosterTarget, OffersTradeBait, OffersWake, OpenBrowser, OpenDraftRoom, OpenLineup, OpenMFLAccounting, PersonalLeagueRecords, PersonalPrincipal, PhotosOff, PlayerName, Principal, RefreshComms, RefreshDraftReports, RefreshMarket, RefreshOffersPool …
- **internal/nativecheck** (81): ActionSwitch, Availability, Award, CanEnterIR, CanEnterTaxi, Check, CheckRoster, ChooseBidWinner, Close, ContractActions, ContractWords, FormatPoints, Known, LateAfter, LateIncrement, LineCardsHeld, Name, NewBidPolicy, Points, PointsText, PriceContract, addedYears, basis, boundsPhrase, buyout, capRelief, charge, compensationWords, countRefusal, countedPlayers, dollars, durationWords, extension, extensionPrice, increaseRefusal, judgePositions, judgeSize, known, lengthRefusal, lowestTier, minimumFor, moveLimit, multipliersUsable, need, note, percentWords, pick, plural, positionLine, priceExtension, priceRelief, priceRestructure, priorityOrder, refuse, release, remainingCells, resign, resignYears, restructure, restructuredNote …
- **internal/leagueservice** (82): Admit, Admit, Admit, Admit, Allow, Blocked, CommissionerAuthorization, CommissionerAuthorization, CommissionerContext, CommissionerContext, CommissionerNow, CommissionerNow, CommissionerPreviewRegistry, CommissionerPreviewRegistry, CommissionerPrincipal, CommissionerPrincipal, CommissionerRecords, CommissionerRecords, CommissionerZone, CommissionerZone, Execute, ExecuteCommissionerWithFacts, ExecuteCommissionerWithFacts, ExecuteRules, FixtureLedger, FixtureRecords, Host, NativeFacts, NativeFacts, NativeFacts, NativeScheduleDates, NativeScheduleDates, NativeScheduleDates, NewFixtureChanges, NewFixtureChangesWithWorker, NewFootballHost, NewHostClock, NewHosted, Now, Pick, PresetChanges, Preview, PreviewCommissionerWithFacts, Ready, Ready, RecordDraftPick, RecordDraftPick, Schedule, SetLineup, TargetCommissionerActor, TargetCommissionerControls, TargetCommissionerLedgerFiltered, TargetCommissionerPresetCompare, TargetLeagueSchedule, TargetNativeBids, TargetNativeContractQuote, TargetNativeFranchiseSchedule, TargetNativeLineupCheck, TargetNativeLineupDelta, TargetNativeMoves …
- **internal/mflref/envelope** (99): Await, Check, ContractInvalidation, ContractPosted, ContractWindow, Created, DraftBid, DraftContract, DraftIR, DraftLineup, DraftRelease, DraftTaxi, DraftTrade, DraftTradeAct, Error, Evaluate, Evaluate, Evaluate, Evaluate, Evaluate, Evaluate, Evaluate, Evaluate, Evaluate, Evaluate, Evaluate, Evaluate, Evaluate, Evaluate, Evaluate, Evaluate, Evaluate, HandOff, HandOffManual, HandOffRelease, ID, Invalidate, IsContract, IsRelease, IsTrade, LineupPositions, MatchesPendingTrade, New, Observe, Rebase, Receipt, ReleaseInvalidation, ReleaseOpen, Restore, Sources, Sources, Sources, Sources, Sources, Sources, Sources, Sources, State, WithdrawBid, WithdrawContract …
- **internal/serverclient** (75): Act, Close, Close, CloseIdleConnections, ConfigureUpdater, DecodeConfirmRequest, Delete, Delete, Dispatch, EncodeConfirmRequest, Error, FixtureChange, FixtureLedger, Get, Get, GoString, Handler, HelperLink, Is, Ledger, NewAgent, NewAgentKeychain, NewDesktop, NewFixture, ParseHelperLink, PracticeUpdate, Prepare, RoundTrip, Set, Set, SetHelperCommands, String, Validate, Validate, Validate, Validate, Validate, Validate, Validate, Validate, Validate, Validate, WatchInvalidations, Workflow, actLocked, authenticator, clearCredential, confirmHelper, consumeInvalidations, decodeDesktop, desktopOrigin, desktopRefusal, enter, helper, installUpdate, launchUpdate, login, newDeviceKeychain, newHTTPClient, post …
- **internal/leaguemigration/mflsource** (57): Adjustments, Draft, DraftType, League, LineupReference, Players, Prepare, Read, Read, ReadCandidate, Results, Rosters, Schedule, Settings, Transactions, adjustments, annotateHistoricalPick, baselineCorrections, baselineRosterCorrections, cap, champions, conferences, contractYear, create, franchise, franchises, heldSelection, known, league, limitRange, lineup, number, originalPickEvidence, owners, picks, playerID, players, position, positionBounds, prepareTransactionPlan, problem, problem, readCandidate, reconstructAsset, reconstructPick, regularWeeks, reverseTransaction, roster, rosterStatus, salary, scoring, snapshot, standings, waiting, waitingSettings, wrapped, year
- **internal/db** (44): AddPlayerCareer, CaptureHosted, CheckAuditAnchor, CheckForeignKeys, CreateLeagueRecovery, CreatePlayerIdentity, CreatePlayerRecord, CreateServerRelease, GuardJSON, GuardPlayerEvidence, Inspect, InstallRosterIdentities, LeagueIdentities, MergePlayerIdentities, NewPlayerIdentity, OpenReadOnly, OpenScratch, ReadLeagueRestoreFile, Restore, SetWriteGuard, StampServerRelease, Validate, adoptMain, baselineAudit, checkpoint, checkpointOffline, copyBackupFile, emptyProfile, fileNames, finish, guardJSONColumn, guardMainJSON, journalStep, moveCurrent, prepareSafety, readHeader, readRestoreJournal, removeIfPresent, rollbackFresh, rollbackMember, sqlIdentifier, stage, swap, validateRollback
- **internal/app/sources** (70): Body, CacheScopeReadings, CacheScopes, CheckArchiveDir, ChecksReport, ClearCache, Commands, CompileAvailable, CompileFacts, CompileFactsWith, CountText, CrosswalkReading, CuratedCount, DataScopes, DataWords, FactsInput, FactsReport, GetCrosswalkReport, HealthReport, HistoryManifest, ImportArchiveFrom, ImportLeagueArchive, LabManifest, LabTime, LeagueRefreshSummary, LoadCrosswalk, PlainAge, Positions, PublishLabManifest, Read, SetLeagueArchiveDir, SetPhotos, SignalFilesReport, SignalsSummary, SignalsView, SnapshotAttempts, SnapshotSignals, TargetCacheClear, TargetCacheScopes, TargetCrosswalk, TargetDataRefresh, TargetDataSources, Thousands, WarmArchive, WriteSettings, checkItemRows, checkProves, checkResultText, compileManifest, coverage, crosswalkShare, exportCounts, factsOutcome, failedLoad, failureText, feedView, itemClass, itemReason, mapsNewMeasure, missingText …
- **internal/app/season** (65): AllPlay, BracketFor, BracketResult, BracketSeeds, BracketSide, BracketWinner, BracketWinnerOf, Brackets, Bubble, ChooseWeek, DirectoryNote, DivisionName, Fields, FindGame, GameResult, LineupDetail, Lineups, Name, NamesInOrder, NeedTeam, NewView, Player, PlayerPoints, PlayoffReport, PositionPoints, Positions, Power, ProjectedBracketSide, Row, ScheduleByTeam, ScheduleByWeek, ScheduleGrid, ScoredSides, SeasonSummary, Seeding, Short, Standings, StandingsOrder, Starters, StepRow, TableKeys, TeamChoices, Tiebreakers, ValueText, WeekChoices, WeekLabel, WeekSides, WeeklyPoints, WeeksReport, aggText, cmpPct, halfGamesText, keyName, mflPower, moveText, pctText, powerHow, powerValues, recordText, scoreText …
- **internal/app/mflref/trades** (55): Accept, Action, BuilderTeams, CentsCorrection, CheckExitPending, ClockReasons, Close, Commands, DeadlineReasons, DealText, Exit, FranchiseName, FuturePickName, Handoff, Keep, LabelReceipts, Legs, PickOwnershipReason, PlanID, PlayerAsset, ProjectedTeam, ReadLineupDelta, Remove, Request, RequireOwner, RichRoster, RosterFigure, Run, Run, Run, Saved, ServiceTeams, SimulationPlayers, TargetDraftTradeAccept, TargetTradeBuilder, TargetTradeBuilderPreview, TargetTradeBuilderRead, Teams, Validate, Validate, WireAssets, WireProjection, WireTeams, builderReading, cloneCell, countRoster, emptyBuilder, executeProjection, heldTradeAssetName, serviceAssets, simulationInput, tradeAssets, tradeBuilder, tradeCapFigures, validateTradeAssets
- **internal/app/research** (40): Commands, PlayerIDs, PublishStatus, ReadNewsBatches, ReadPlayerUsage, RefreshReading, RosteredAssetIDs, Run, Run, SnapshotStatus, TargetOpenResearchArticle, TargetPlayerUsage, TargetResearchNewsRefresh, TargetResearchRefresh, TargetResearchStatus, UsageMeasures, add, denominator, depthGrid, fallbackResearchNews, floatValue, indexUsage, isolatedNews, latestTeam, measures, newsPosted, nextOpponent, readFeed, researchNewsAge, shares, shortDepthNames, teams, total, usageLabel, usageMatchup, usageMeasure, usageRank, usageSnapMeasure, value, window
- **internal/app/offers** (68): CapRoom, Commands, MFLCommands, MarketLines, PersonalSnapshot, PoolNumber, PoolSorts, SortPool, TargetBidDraft, TargetBidHandOff, TargetBidPlans, TargetBidPosted, TargetBidPreview, TargetBidRevise, TargetBidWithdraw, TargetNativePersonalChange, TargetNativePersonalNotes, TargetPersonal, TargetPersonalChange, TargetPersonalHandOff, TargetPersonalNotes, TargetPool, TargetPoolRefresh, Validate, Validate, Validate, Validate, Validate, Validate, activeBids, amountOf, beatText, bidActions, bidActive, bidConditional, bidDropOptions, bidInput, bidMove, bidOpen, bidPlan, bidPlans, bidStateWord, boardSummary, closesNote, entry, formOf, loadBid, nativePersonalScope, noNFLTeam, note, noted, openBidPlansByPlayer, personalPlayers, personalSnapshot, pointsText, poolMarketValue, poolOrder, poolReading, poolRow, poolRows …
- **internal/governance** (91): AccountPrincipal, AppealJudge, Audience, Checkpoint, CoCommissioners, ElectionStatus, ElectionThreshold, InactivityEligible, Invite, LastActivity, League, NewPrincipals, Private, Private, Private, Private, Private, Private, Private, Private, Private, Private, Private, Private, Private, Private, Private, Private, Private, Private, Private, Private, Private, Private, Private, Private, Private, Private, Resume, Reviewers, SuccessionPeriods, TradeReviewReady, Validate, Vote, VoteOutThreshold, addListed, advanceSuccession, allowOwner, apply, apply, apply, apply, apply, apply, apply, apply, apply, apply, apply, apply …
- **internal/leaguemigration** (47): AsProblems, Ask, Base, CarriedFields, Compare, Confirm, Cutover, DefaultPreset, Error, Error, Import, ImportWithOptions, Invites, Prepare, ReadApp, Review, Review, SendInvites, Unwrap, Validate, Words, add, answerCommands, await, blankCitations, charges, checkScheduleMapping, describe, franchiseCheck, franchises, group, inviteID, invites, league, pickKey, picks, planTransactionCommands, players, prepareInvites, prepared, read, read, review, settings, stateAndValue, syncImportedReferences, validEmail
- **internal/store/accounts** (36): AuthGeneration, BeginTOTP, BindAlias, CheckRecentAuth, ConsumeTOTP, CreateSession, Credential, Insert, IssueEnrollment, IssueToken, LastResets, RecentAuthRemaining, RedeemEnrollment, ReplaceRecoveryCodes, ResetAccount, ResetAfterRestore, ResetSignIns, RevokeSession, RevokeToken, SeatStatus, SessionAccount, SetClock, TOTP, Token, VerifyAudit, VerifyAuditAnchor, accountIDs, appendSecurityAudit, lastReset, readSession, recoveryAudit, resetAuth, resetCredentials, validSecretDigest, validateEnrollment, validateTOTPAccount
- **internal/leaguerules** (45): Default32, Default32ForTeams, Default32Scoring, Diff, DropTotals, DurationWords, EvenDivisions, Fill, FromMFLScoring, LegacyNFLForTeams, LowestMinimumSalary, Mirror16, Mirror16ForTeams, SeedChanges, StartupRoundsFor, StatByMFL, StatDictionary, Status, UnmarshalJSON, UnmarshalJSON, UnusedStat, Validate, cutRange, decodeKey, decodeList, decodeMap, default32Alignment, default32Conferences, default32Divisions, default32EdgeRows, default32Extras, default32LineUnit, default32Rows, default32Scales, difference, kindWords, legacyCatalogueBeforeKinds, mflDecimal, mflPositions, mflRule, mirror16Alignment, mirror16Graph, ruleSetting, sameStored, startupValues
- **internal/app/control** (55): AccountingReport, AccountingURL, BylawSections, ClearPending, Commands, CompareAlerts, Delete, DeleteMFLKey, FlattenScheduleWeeks, InjuryReturns, LeagueSettingsReport, OffersTitle, OpenAccounting, PendingParent, PendingRefreshBackground, PublishRuleNames, PublishSiteNews, ResetParam, ResetParameter, Set, Set, SetMFLKey, SetParam, SetParameter, SiteNewsReport, SnapshotRuleNames, SnapshotSiteNews, TargetAccounting, TargetBylaws, TargetLeagueSettings, TargetMFLMigration, TargetOpenMFLAccounting, TargetOpenMFLLineup, TargetSetWindow, TargetSiteNews, Validate, Validate, bylawLine, capRows, cloneRuleNames, cloneSiteNews, isTableRule, limitsText, percentText, plainMarkdown, ruleName, scoresNothing, scoringPoints, scoringText, setCalendarWindow, setTradeWindow, startText, tiebreak, tiebreakText, weeksThrough
- **internal/accounts** (97): Admission, Admission, Admit, Admit, Authenticate, Authenticate, Authenticate, Begin, CanonicalUsername, Check, CheckPrivileged, Confirm, ConsoleRecover, DecodePasswordHash, DelegableCommand, GoString, GoString, GoString, GoString, GoString, GoString, GoString, GoString, GoString, GoString, GoString, GoString, GoString, GoString, Hash, HashPassword, Issue, Issue, Lifetime, LocalBootstrap, Login, MarshalJSON, MarshalJSON, MarshalJSON, MarshalJSON, NewEnrollment, NewEnrollmentWithPasswordWork, NewRecovery, NewSessions, NewTOTP, NewTokens, Principal, PrintCodes, PrivilegedRemaining, Recover, Redeem, Revoke, Revoke, SignIn, Status, StepUp, String, String, String, String …
- **internal/ingestion/livegame** (34): GameLive, ParseCore, ParseSummary, ParseYahoo, ScoringPlays, Validate, Validate, Validate, Validate, Validate, Validate, Validate, Validate, Validate, Validate, boxCategory, delay, eventID, fetchBoard, fetchDetail, fetchFinal, holdFinal, keptFinal, play, rawPlays, referenceID, refreshBoard, refreshDetail, sameGame, state, validatePlays, validateScoreClock, withPlays, yahooState
- **internal/transactions** (104): Error, Execute, Kind, Kind, Kind, Kind, Kind, Kind, Kind, Kind, Kind, Kind, Kind, Kind, Kind, Kind, Kind, Kind, Kind, Kind, LegalOps, PositionFloor, Preview, ProjectContract, allPhases, appendWith, apply, apply, apply, apply, apply, apply, apply, apply, apply, apply, apply, apply, apply, apply, apply, apply, apply, apply, checkPositionLimit, checkRosterSize, checkRosterSizeFromNet, countByStatus, deadCapDeltas, enforceRosterLimits, enforceRosterLimits, enforceRosterLimits, enforceTradeFranchiseLimits, gatePhase, inTx, phaseAllowed, phasePolicy, positionCount, resolve, resolve …
- **internal/fantasyscore/source** (31): ConsolidateFinal, Convert, Convert, Convert, FillStatGaps, LineGames, LiveAuthority, LiveRevisionDigest, LiveSleeperLines, MergeStatSources, ReadLinePlays, ReadLineRushing, SelectHeldWeek, Validate, Validate, Validate, attachBeforeContact, countLinePlay, countRush, fields, fillSleeperBands, finalAuthorityLine, linePlayColumns, optionalDecimal, sleeperBands, sleeperBinsReconcile, sleeperDefenseFields, sleeperEdgesFields, sleeperFieldMap, sleeperKickingFields, sleeperOffenseFields
- **internal/engine/rookiedraft** (44): Available, Error, FirstRound, NextPick, PreviewSelection, Replay, ReplayActs, ReplaySideActs, ResolveSetup, ResolveStartup, SlotTerms, StartupFirstRound, StartupPicks, StartupRules, admits, advance, apply, change, choose, clockAt, fill, find, insert, newTimeline, onClock, open, ownerPlayers, placeOwners, preparedPicks, refuseSelect, replayActs, rookie, rosterOverage, seal, settle, skip, slotSalary, startupRounds, startupSalary, timeoutPolicy, timer, validateOrder, validateReplayRound, validateRound
- **internal/engine/competition** (34): BuildBracket, GameID, GenerateSchedule, LeagueWide, advanced, alignedScheduleRounds, balanceScheduleHomes, crossDivisionRounds, decide, fill, finalWords, followed, fromPool, needsRuling, newBracketBuilder, order, orientScheduleEdge, orientScheduleTrails, pass, reachNext, resolve, round, roundName, scheduleDivisionGroups, scheduleOrder, scheduleRoundScore, scheduleRounds, scheduleTrailStart, scopes, seeded, settle, side, unplaced, waitingWords
- **internal/app/market** (30): ArchiveScope, BuildBoard, Commands, DynastyCells, Facts, NativePickHistory, OnFieldNow, ParseArchiveFeed, PickTotals, PickValue, Player, PoolAssetValues, ReadAgeArc, RefreshReports, Signed, SnapshotModel, StoredParams, TargetAgeArc, TargetAssetBoard, TargetMarketReportsRefresh, TopPercent, ValueCells, WireAssetBoard, buildAgeArc, draftSlots, fact, readArcPlayer, rookieCapital, runParamSet, wireSlice
- **internal/commandcatalog** (17): Entries, Invoke, NewOperatorRegistry, Publication, RunMigration, Validate, Validate, argumentFields, callBinding, decodeJSON, handlers, request, result, runWorkflowChange, runWorkflowPick, validateWorkflowChange, validateWorkflowPick
- **internal/app/platform** (21): Commands, DatabaseChecksReading, DelegableCommands, IDs, List, Manual, NativeUpdateLock, Photo, Probe, Restore, RestoreFailure, RunFlag, RunWindow, SmokeWindow, TargetGateAudit, TargetInjuries, TargetPhoto, Validate, databaseCheckWords, recordHeapProfiles, writeHeapProfile
- **internal/app/draft** (30): Build, Commands, Destination, NFL, Names, NextDate, NextDraft, OpenRoom, ParsePlayers, RefreshReports, TargetDraftReports, TargetDraftReportsRefresh, TargetNFLDraft, TargetOpenMFLDraftRoom, ValueNote, Values, completedDraft, draftDuration, draftElapsed, draftFranchises, draftFuture, draftGrid, draftPace, draftStatus, draftValueReport, orderedPicks, pickLabel, pickPlayer, pickTime, shell
- **internal/app/mflref/planwatch** (28): AuthorizeTeam, Decision, Draft, HandOff, MoveLog, MovePredicate, Read, ReadMoveObservation, Recent, RequestMovesPass, ReturnToDraft, RosterMoveTarget, RosterRail, Run, Run, SaveMoveEvidence, SkipMoveRefresh, Sources, TargetCheckMoves, TargetDraftIR, TargetDraftTaxi, TargetHandOff, TargetMoves, TargetWithdrawRoster, ValidateMoveURL, WatchableMove, WithBidFacts, Withdraw
- **internal/ingestion/leaguearchive** (28): Check, HeldKey, Import, Key, NewDiskFS, Open, ParsePath, ParseURL, ReadDir, ReadFile, ScopeOf, URL, Validate, Validate, Week, checkBody, compress, fixedParams, openChecked, ownerExports, parentDomain, parseName, pathInfo, readArchive, readFile, readLeagues, siteExports, suffixParam
- **internal/releasetrust** (25): DecodeConfig, Download, DownloadFile, New, Stage, Validate, Validate, Validate, Validate, VerifyServerBundle, admitRelease, atomicUpdateFile, decodeRelease, protectServerCache, publicName, publicNameCharacter, publicRedirect, readLimited, repositoryPath, safeTarget, serverRelease, targetCharacter, updater, validateRoot, verifyBundleBytes
- **internal/app/comms** (21): Build, Commands, Read, RefreshReading, RefuseBoards, RosterWeekTimes, TargetComms, TargetCommsRefresh, TargetOpenComms, URL, Validate, asset, assets, commsKind, commsMatches, commsPosts, contract, droppedContract, team, transactionSentence, wireComms
