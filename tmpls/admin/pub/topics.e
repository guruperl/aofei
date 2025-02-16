[% INCLUDE start.e %]

      <div align="center">
        <table border="1" id="tblGrid" class="tblGrid">
          <thead>
            <tr>
            <th>ID</th>  
            <th>Login</th>
              <th>Company</th>
              <th>Contact</th>
              <th>Phone</th>
              <th>Created</th>
              <th>Status</th>    
              <th> </th>
            </tr>
          </thead>
          <tbody>



[% FOREACH row IN topics %]
  <tr id="[% GET row.pubid %]" class="userRow">
    <td style="text-align: left"><a href="site?action=topics&pubid=[% GET row.pubid %]&company_esc=[% row.company_esc %]">[% GET row.pubid %]</a></td>
    <td>[% row.email %]</td>    
    <td nowrap><a target=_blank href='[% GET row.url %]'>[% GET row.company %]</a></td>
    <td>[% GET row.contact %]</td>        
    <td>[% GET row.phone %]</td>      
    <td>[% GET row.created %]</td>               
    <td style="text-align: center"><img src='/uilib/comImg/[% row.status %].png' /></td>
    <td class="userData">
      <a href="pub?action=edit&pubid=[% GET row.pubid %]"><img src="/uilib/comImg/editor.gif" border=0 alt="Edit" /></a>
      <a href="pub?action=editpass&pubid=[% GET row.pubid %]"><img src="/uilib/comImg/passwd.jpg" border=0 alt="Edit Password" /></a>
      <a href="javascript:execConfirmHrefDelete('./pub?action=delete&pubid=[% GET row.pubid %]','e')"><img src="/uilib/comImg/delete.gif" border=0 alt="Delete" /></a>
    </td>
  </tr>  
[% END %]

[% INCLUDE end.e %]
