package observability

/*
RemoteObservability is used to get feedback on part of the loading process.
In particular when we load repos, teams, and users.
It is mostly used for UX purposes (on the plan)
*/
type RemoteObservability interface {
	// Init is called to specify the number of assets we will be loading
	Init(nbTotalAssets int)
	// LoadingAsset is called when we start loading a github asset
	LoadingAsset(entity string, nb int)
}
