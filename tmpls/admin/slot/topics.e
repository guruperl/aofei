<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
  <head>
    <title>Slot Management</title>
    <script src="../../../js/jquery-1.4.2.min.js"></script>
    [% INCLUDE topics_ui.e %]
  </head>
  <body>
   
    <div align="center">
       <h2 align="center" class="curTitle"><a href='pub'>Publishers</a> : <a href='site?action=topics&pubid=[% pubid %]&company_esc=[% company_esc %]'>[% company %]</a> : [% sitename %] > Slot Management</h2>
<div class="navContainer">
      <div class="navList">
        <a id="logout" href="../../../go.cgi?action=logout&role=admin">Log out</a>
<p></p>
      </div>
</div>       
      <table id="tblGrid" border="1" class="tblGrid">
        <thead>
          <tr>
            <th>Slot&nbsp;ID</th>
            <th>Slot&nbsp;Name</th>
            <th>Size&nbsp;ID</th>            
            <th>Created</th>
            <th>Status</th>
            <th>Actions</th>
          </tr>  
        </thead>
        <tbody>
          [% FOREACH row IN topics %]
            <tr id="[% GET row.siteid %]_[% GET row.slotid %]" class="Slot">
              <td><a href="weight?action=topics&slotid=[% GET row.slotid %]&slotname_esc=[% row.slotname_esc %]&siteid=[% siteid %]&sitename_esc=[% sitename_esc %]&pubid=[% pubid %]&company_esc=[% company_esc %]">[% GET row.slotid %]</a></td>
              <td>[% GET row.slotname %]</td>
              <td>[% GET row.sizeid %]</td>
              <td>[% row.created %]</td>
              <td><img src='/uilib/comImg/[% GET row.status %].png' /></td>
              <td>
                <a href="slot?action=edit&slotid=[% GET row.slotid %]&siteid=[% GET row.siteid %]"><img src="/uilib/comImg/editor.gif" border=0 alt="Edit" /></a>&nbsp;&nbsp;
                <a href="javascript:execConfirmHrefDelete('creative?action=delete&slotid=[% GET row.slotid %]&siteid=[% GET row.siteid %]','e')"><img src="/uilib/comImg/delete.gif" border=0 alt="Delete" /></a>
              </td>            
            </tr>
          [% END %]
        </tbody>
      </table>
      
    </div>  
  </body>
</html>
