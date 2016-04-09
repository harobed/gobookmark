$(document).ready(function() {
  console.log("fooobar");
  $('input[name="url"]').bind("propertychange change click keyup input paste", function() {
    $.get("/fetch-title/?url=" + $('input[name="url"]').val(), function(data) {
      if (data != '') {
        $('input[name="title"]').val(data);
      }
    });
  });
});
