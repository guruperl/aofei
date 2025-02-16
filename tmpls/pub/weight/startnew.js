ITEMS = [ [% FOREACH item=startnew %]
	{[% FOREACH pair IN item.pairs %]"[% pair.key %]" : "[% pair.value %]", [% END %]}, [% END %]
];
