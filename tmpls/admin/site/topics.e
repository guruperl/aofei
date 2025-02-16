[% INCLUDE start.e %]

      <div align="center">
        <table id="tblGrid" border="1" class="tblGrid">
          <thead>
            <tr>
              <th>Site&nbsp;ID</th>
              <th>Site&nbsp;Name</th>
             <th>Priority</th>
             <th>Quality&nbsp;Score</th>
			<th>Created</th>
              <th style="text-align: center">Status</th>
              <th> </th>
            </tr>
          </thead>
          <tbody>

[% FOREACH row IN topics %]
  <tr id="[% GET row.pubid %]_[% GET row.siteid %]" class="Row">
    <td>
      <a href="slot?action=topics&siteid=[% GET row.siteid %]&sitename_esc=[% row.sitename_esc %]&pubid=[% pubid %]&company_esc=[% company_esc %]">[% GET row.siteid %]</a></div>
    </td>
    <td nowrap><a target=_blank href='[% GET row.siteurl %]'>[% GET row.sitename %]</a></td>
    <td>[% GET row.priority %]</td>
    <td>[% GET row.qa_site %]</td>
    <td>[% GET row.created %]</td>
    <td style="text-align: center"><img src='/uilib/comImg/[% GET row.status %].png' /></td>
    <td class="rowData">
    <a href="site?action=edit&pubid=[% GET row.pubid %]&siteid=[% GET row.siteid %]"><img src="/uilib/comImg/editor.gif" border=0 alt="Edit" /></a>&nbsp;&nbsp;
      <a href="javascript:execConfirmHrefDelete('site?action=delete&siteid=[% GET row.siteid %]&pubid=[% GET row.pubid %]','e')"><img src="/uilib/comImg/delete.gif" border=0 alt="Delete" /></a>
    </td> 
  </tr>  
[% END %]

[% INCLUDE end.e %]
