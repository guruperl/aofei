// Package acl models the supply-side hierarchy used by the bid path.
//
// A Pub owns Sites, and a Site owns Slots. Runtime matching starts from the
// publisher domain on the bid endpoint, resolves the Pub/Site/Slot cache entry,
// and carries that RPub identity through targeting, response generation, and
// win/loss accounting. ACL audiences apply allow/deny rules for that supply
// identity before a campaign can win the impression.
package acl
