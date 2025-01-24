package observability

/*
RemoteLoadFeedback is used to get feedback on part of the loading process.
In particular when we load repos, teams, and users.
It is mostly used for UX purposes (on the plan)
*/
type RemoteLoadFeedback interface {
	LoadingAsset(nb int)
}
