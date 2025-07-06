module.exports = {
  content: ["./src/**/*.{js,jsx,ts,tsx}"],
  theme: {
    extend: {
      keyframes: {
        fadein: {
          "0%": { opacity: "0", transform: "translateY(20px)" },
          "100%": { opacity: "1", transform: "translateY(0)" }
        }
      },
      animation: {
        fadein: "fadein 0.9s ease-out forwards"
      }
    }
  },
  plugins: [
    require("tailwindcss-animation-delay")
  ]
};