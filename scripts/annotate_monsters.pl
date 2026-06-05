#!/usr/bin/perl
# Annotates the Draw Steel Monsters source with @type comments by structural
# position. Reads the clean (unannotated) source on STDIN, writes annotated
# markdown to STDOUT. Frontmatter is prepended by the caller.
use strict;
use warnings;
use utf8;
binmode(STDIN, ':encoding(UTF-8)');
binmode(STDOUT, ':encoding(UTF-8)');

my @lines = map { my $l = $_; $l =~ s/\n$//; $l } <STDIN>;

# slug() mirrors Go's content.Slugify: lowercase, drop apostrophes, non-alnum
# runs -> '-', trim leading/trailing '-'.
sub slug {
    my $s = lc shift;
    $s =~ s/['\x{2019}]//g;
    $s =~ s/[^a-z0-9]+/-/g;
    $s =~ s/^-+//;
    $s =~ s/-+$//;
    return $s;
}

# --- Pass 1: detect which Dynamic Terrain H3s are categories (have H9 children).
my %is_terrain_category;
{
    my $chapter = '';
    my $last_h3 = -1;
    for my $i (0 .. $#lines) {
        next unless $lines[$i] =~ /^(#+)\s+(.+?)\s*$/;
        my $n = length($1);
        if ($n == 1) { $chapter = $2; $last_h3 = -1; next; }
        next unless $chapter eq 'Dynamic Terrain';
        if    ($n == 3) { $last_h3 = $i; }
        elsif ($n == 9) { $is_terrain_category{$last_h3} = 1 if $last_h3 >= 0; }
    }
}

# --- Pass 2: emit annotations.
my %chapter_id = (
    'Monster Basics'  => 'monster-basics',
    'Monsters'        => 'monsters',
    'Dynamic Terrain' => 'dynamic-terrain',
    'Retainers'       => 'retainers',
);

my $chapter = '';
my $out = '';
for my $i (0 .. $#lines) {
    my $line = $lines[$i];
    my $ann;

    if ($line =~ /^(#+)\s+(.+?)\s*$/) {
        my $n     = length($1);
        my $title = $2;

        if ($n == 1) {
            $chapter = $title;
            $ann = "<!-- \@type: chapter | \@id: $chapter_id{$title} -->" if exists $chapter_id{$title};
        }
        elsif ($n == 2 && $chapter eq 'Monsters') {
            $ann = "<!-- \@type: monster | \@category: " . slug($title) . " -->";
        }
        elsif ($n == 3 && $chapter eq 'Monsters' && $title =~ /(\d(?:st|nd|rd|th))\s+Echelon/i) {
            # Echelon sub-groups (Rivals/Demons/Undead/War Dogs) — add an echelon
            # path segment so repeated statblock names stay distinct.
            $ann = "<!-- \@type: monster-group | \@subcategory: " . lc($1) . "-echelon -->";
        }
        elsif ($n == 3 && $chapter eq 'Dynamic Terrain' && $is_terrain_category{$i}) {
            $ann = "<!-- \@type: monster-group | \@domain: dynamic-terrain | \@category: " . slug($title) . " -->";
        }
        elsif ($n == 4 && $chapter eq 'Retainers' && $title eq 'Retainer Statblocks') {
            $ann = "<!-- \@type: monster-group | \@domain: retainer -->";
        }
        elsif ($n == 4 && $chapter eq 'Monster Basics' && $title eq 'Harmless Creatures') {
            $ann = "<!-- \@type: monster-group | \@category: noncombatant -->";
        }
        elsif ($n == 7) {
            # Every H7 is a statblock (Monsters, Retainers, and the lone
            # Noncombatant in Monster Basics).
            $ann = "<!-- \@type: statblock -->"
                if $chapter eq 'Monsters' || $chapter eq 'Retainers' || $chapter eq 'Monster Basics';
        }
        elsif ($n == 9) {
            if    ($chapter eq 'Monsters')        { $ann = "<!-- \@type: featureblock -->"; }
            elsif ($chapter eq 'Dynamic Terrain') { $ann = "<!-- \@type: dynamic-terrain -->"; }
            # Monster Basics H9 (Basic Malice) is left unannotated, matching legacy.
        }
    }

    $out .= "$ann\n" if defined $ann;
    $out .= "$line\n";
}
print $out;
