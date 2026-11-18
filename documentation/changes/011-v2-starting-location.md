# v2 starting location

Add an optional starting saved place to optimization previews. Day start means
departure from that place; include its directed leg to the first activity in
feasibility and travel savings. No return leg. The starting place may also be an
activity and must belong to the owned trip. Preserve every activity.

Fetch one matrix with the origin appended, then separate its outbound vector
from activity-to-activity legs. Max 20 activities + 1 origin = 441 elements,
within the provider's general matrix bound. Existing per-user/global quotas
still charge all requested elements. UI default remains start at first activity.

Test origin-driven reordering, unavailable first leg, invalid vectors, foreign
saved place rejection, and browser persistence with an explicit starting place.
No schema migration or free-text geocoding. Document before implementation.

Implemented optional origin vectors through evaluation and proposal generation,
provider matrix capacity of 21 locations, owned saved-place lookup, and UI choice.
Go race/PostgreSQL tests, vet, frontend typecheck and lint passed. Fixtures verify
origin-driven ordering, included first-leg time/cost, unavailable origin routes,
and invalid vectors/foreign starting places. Full browser coverage follows with
remaining v2 integration work.
