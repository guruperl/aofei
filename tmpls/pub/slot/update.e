[% INCLUDE start.e %]

<div class="ui-layout-west">
		<ul>
        <li><a href="site?action=edit&siteid=[% siteid %]">[% sitename %]</a>
		<p></p>
			<ul>
            <li><a href="slot?action=edit&slotid=[% slotid %]&siteid=[% siteid %]&sitemd5=[% sitemd5 %]&sitename_esc=[% sitename_esc %]">[% slotname %]</a></li>
        </ul></li>
</ul>
</div>
<div class="ui-layout-center">

updated.

</div>

[% INCLUDE end.e %]
