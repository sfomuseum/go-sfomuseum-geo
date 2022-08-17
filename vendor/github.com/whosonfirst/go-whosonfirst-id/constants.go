package id

// MULTIPLE_PARENTS is the identifier used to indicate that a place is legitimately parented by multiple (other) places.
const MULTIPLE_PARENTS = -4

// ITS_COMPLICATED is the identifier used to indicate that the parentage of a place is complicated in a geopolitical way too nuanced and complex to express otherwise.
const ITS_COMPLICATED = -2

// UNKNOWN is the identifier used to indicate that an otherwise valid identifier is unknown and needs to be resolved.
const UNKNOWN int64 = -1

// EARTH is the Who's On First identifier for the planet Earth.
const EARTH int64 = 0

// NULL_ISLAND is the Who's On First identifier for the Null Island.
const NULL_ISLAND int64 = 1
