function submit_form()
{
    var data = {
        "serverurl" : $("#serverurl").val(),
        "username" : $("#username").val(),
        "password" : $("#password").val(),
        "silent" : $("#silent").prop("checked") ? 1 : 0,
        "append_rnd" : $("#append_rnd").prop("checked") ? 1 : 0,
        "clientname_prefix": $("#clientname_prefix").val(),
        "notray" : $("#notray").prop("checked") ? 1 : 0,
        "group_name": $("#group_name").val(),
        "sel_os": $("#sel_os").val(),
        "retry": $("#retry").prop("checked") ? 1 : 0
    };

    $("#data").val(JSON.stringify(data));
}