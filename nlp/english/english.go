package english

import (
	"strings"

	"github.com/future-architect/watertower/nlp"
	"github.com/kljensen/snowball/english"
)

const Language = "en"

func init() {
	stopWords := make(map[string]bool)
	/*for _, stopWord := range strings.Fields(stopWordsSrc) {
		stopWords[stopWord] = true
	}*/
	nlp.RegisterTokenizer(Language, englishSplitter, englishStemmer, stopWords)
}

func englishSplitter(content string) []string {
	words := strings.Fields(content)
	result := make([]string, 0, len(words))
	for _, word := range words {
		word = strings.TrimRight(word, ".,:\"")
		result = append(result, strings.ToLower(word))
	}
	return result
}

func englishStemmer(content string) string {
	return english.Stem(content, false)
}

// https://github.com/stopwords-iso/stopwords-en/blob/master/raw/snowball-tartarus.txt
var stopWordsSrc = `
i
me
my
myself
we
us
our
ours
ourselves
you
your
yours
yourself
yourselves
he
him
his
himself
she
her
hers
herself
it
its
itself
they
them
their
theirs
themselves
what
which
who
whom
this
that
these
those
am
is
are
was
were
be
been
being
have
has
had
having
do
does
did
doing
will
would
shall
should
can
could
may
might
must
ought
i'm
you're
he's
she's
it's
we're
they're
i've
you've
we've
they've
i'd
you'd
he'd
she'd
we'd
they'd
i'll
you'll
he'll
she'll
we'll
they'll
isn't
aren't
wasn't
weren't
hasn't
haven't
hadn't
doesn't
don't
didn't
won't
wouldn't
shan't
shouldn't
can't
cannot
couldn't
mustn't
let's
that's
who's
what's
here's
there's
when's
where's
why's
how's
daren't
needn't
doubtful
oughtn't
mightn't
a
an
the
and
but
if
or
because
as
until
while
of
at
by
for
with
about
against
between
into
through
during
before
after
above
below
to
from
up
down
in
out
on
off
over
under
again
further
then
once
here
there
when
where
why
how
all
any
both
each
few
more
most
other
some
such
no
nor
not
only
own
same
so
than
too
very
one
every
least
less
many
now
ever
never
say
says
said
also
get
go
goes
just
made
make
put
see
seen
whether
like
well
back
even
still
way
take
since
another
however
two
three
four
five
first
second
new
old
high
long
    `
