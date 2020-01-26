
var app = new Vue({
    el: '#app',
    data: {
        username: '',
        email: '',
        subdomain: '',
        listUser: [],
        selected: '',
        editModalShow: false,
        deleteModalShow: false,
    },
    mounted () {
        this.fetchData();
    },
    methods: {
        toggleEdit: function() {
            this.editModalShow = !this.editModalShow;
        },
        toggleDelete: function() {
            this.deleteModalShow = !this.deleteModalShow;
        },
        fetchData: function() {
            axios
            .get('http://localhost:3000')
            .then(response => {this.listUser = response.data; console.log(this.listUser);})
        },
        submit: function() {
            axios.post('http://localhost:3000/create', {
                username: this.username,
                email: this.email,
                subdomain: this.username
            })
            .then(function (response) {
                this.fetchData();
            })
            .catch(function (error) {
                console.log(error);
            });
        },
        editModal: function(user) {
            this.selected = user;
            this.toggleEdit();
        },
        deleteModal: function(user) {
            this.selected = user;
            this.toggleDelete();
        },
        updateBtn: function() {
            axios.post('http://localhost:3000/update', {
                username: this.selected.username,
                subdomain: this.subdomain
            })
            .then(function (response) {
                this.fetchData();
                this.toggleEdit();
            })
        },
        deleteBtn: function() {
            axios.post('http://localhost:3000/delete', {
                username: this.selected.username
            })
            .then(function (response) {
                this.fetchData();
                this.toggleDelete();
            })
        }
    }
});
