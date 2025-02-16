[% INCLUDE start.e %]

<div class="ui-layout-west" style="display:none;">
<ul id="treeList">
        <li><a href="campaign?action=edit&campaignid=[% campaignid %]">[% campaignname %]</a>
			<p></p>
            <ul>
            <li><a href="item?action=edit&itemid=[% itemid %]&campaignid=[% campaignid %]&campaignmd5=[% campaignmd5 %]&campaignname_esc=[% campaignname_esc %]">[% itemname %]</a></li>
			</ul></li>
</ul>
</div>
<div class="ui-layout-center">

item updated.

</div>

[% INCLUDE end.e %]
