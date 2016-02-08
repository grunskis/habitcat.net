function ajax(method, path, successCallback, errorCallback) {
    var req = new XMLHttpRequest();

    req.onreadystatechange = function() {
        if (req.readyState == XMLHttpRequest.DONE) {
            if (req.status == 200) {
                successCallback(req.responseText);
            } else {
                errorCallback(req.status);
            }
        }
    }

    req.open(method, path, true);
    req.send();
}

function updateActivityProgress(uuid) {
    ajax("POST", "/update/" + uuid, function (body) {
        var progress = document.getElementById("done-" + uuid);
        progress.style.width = body + "%";
    }, function (statusCode, body) {
        console.log("fail", statusCode, body);
    });

    return false;
}

function updateHabitProgress(uuid) {
    ajax("POST", "/habits/" + uuid, function (body) {
        var progress = document.getElementById("done-" + uuid);
        progress.style.width = body + "%";
    }, function (statusCode, body) {
        console.log("fail", statusCode, body);
    });

    return false;
}
