
class UserDetails extends HTMLElement {
    static observedAttributes = ["format"];
    #userId;

    constructor() {
        super();
    }

    connectedCallback() {
        this.#userId = this.textContent
        this.render().then(r => console.log(`Tag content replaced with "${r}"`))
    }

    async render() {
        const response = await fetch(`{{ .API_URL }}${this.#userId}`);
        if (response.ok) {
            const data = await response.json();
            const displayText = `${data['family_name']}, ${data['given_name']}, ${data['department']}`
            const myContainer = document.createTextNode(displayText);
            this.replaceChildren(myContainer);
            return displayText;
        } else {
            console.error(response);
        }
        return false;
    }
}


customElements.define("user-details", UserDetails);
